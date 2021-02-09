package tmcevents

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

//TmcEvents defines field needed
type TmcEvents struct {
	TmcHostname     string   `toml:"tmc_hostname"`
	CspHostname     string   `toml:"csp_hostname"`
	CspToken        string   `toml:"csp_token"`
	Events          []string `toml:"events"`
	tokenResponse   *tokenResponse
	tokenExpiration time.Time

	Log telegraf.Logger `toml:"-"`
}

type tokenResponse struct {
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func init() {
	inputs.Add("tmcevents", func() telegraf.Input {
		return &TmcEvents{
			TmcHostname:   "",
			Events:        []string{},
			CspHostname:   "console.cloud.vmware.com",
			CspToken:      "",
			tokenResponse: &tokenResponse{},
		}
	})
}

//Init inits the plugins
func (r *TmcEvents) Init() error {
	return nil
}

//SampleConfig returns the sample plugin configuration
func (r *TmcEvents) SampleConfig() string {
	return `
  ## connects to TMC event stream
	[inputs.tmcevents]
	# tmc_hostname = "value"
	# events = ""
	# csp_hostname = ""
	# csp_token = ""
`
}

//isExpired checks if the token is expired
func isExpired(tokenExpiry time.Time) bool {
	// refresh at half token life
	now := time.Now().Unix()
	halfDur := -time.Duration((tokenExpiry.Unix()-now)/2) * time.Second
	if tokenExpiry.Add(halfDur).Unix() < now {
		return true
	}
	return false
}

//login checks the tokens expiration and creates a new one if neccessary
func (r *TmcEvents) login() error {
	if r.tokenResponse.AccessToken == "" || isExpired(r.tokenExpiration) {
		data := url.Values{}
		data.Set("refresh_token", r.CspToken)
		r.Log.Debug("getting access token")
		resp, err := http.PostForm(fmt.Sprintf("https://%s/csp/gateway/am/api/auth/api-tokens/authorize", r.CspHostname), data)
		if err == nil {
			defer resp.Body.Close()
		} else {
			r.Log.Error(fmt.Sprintf("Call to CSP to authorize refresh token failed with error: %s", err.Error()))
			return err
		}
		respJson, err := ioutil.ReadAll(resp.Body)
		err = json.Unmarshal(respJson, &r.tokenResponse)
		r.tokenExpiration = time.Now().Local().Add(time.Second * time.Duration(r.tokenResponse.ExpiresIn))
	}
	return nil
}

//Gather connects to the TMC api to get the stream of events, it will stay open for a period of time and close when no events
// are present, the telegraf plugin will re-run this command constantly
func (r *TmcEvents) Gather(a telegraf.Accumulator) error {
	_ = r.login()
	var result map[string]interface{}
	url := fmt.Sprintf("https://%s/v1alpha1/events/stream", r.TmcHostname)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.tokenResponse.AccessToken))
	q := req.URL.Query()
	for _, s := range r.Events {
		q.Add("eventTypes", s)
	}
	req.URL.RawQuery = q.Encode()
	r.Log.Debug(req.URL.String())
	res, err := client.Do(req)
	if err != nil {
		return err
		r.Log.Error(err.Error())
	}
	defer res.Body.Close()

	reader := bufio.NewReader(res.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return err
			r.Log.Error(err.Error())
		}
		json.Unmarshal(line, &result)
		r.Log.Debug(result)
		eventresult := result["result"].(map[string]interface{})
		flattenedevent := Flatten(eventresult)
		timefield := flattenedevent["event.time"].(string)

		layout := "2006-01-02T15:04:05.000000000Z"
		timestamp, err := time.Parse(layout, timefield)
		if err != nil {
			fmt.Println(err)
		}
		a.AddFields("event", flattenedevent, nil, timestamp)
	}

	return nil
}

// Flatten takes a map and returns a new one where nested maps are replaced
// by dot-delimited keys.
func Flatten(m map[string]interface{}) map[string]interface{} {
	o := make(map[string]interface{})
	for k, v := range m {
		switch child := v.(type) {
		case map[string]interface{}:
			nm := Flatten(child)
			for nk, nv := range nm {
				o[k+"."+nk] = nv
			}
		default:
			o[k] = v
		}
	}
	return o
}

//Description returns plugin desription
func (r *TmcEvents) Description() string {
	return "Connects to the tmc events api stream"
}
