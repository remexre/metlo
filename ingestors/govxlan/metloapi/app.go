package metloapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/metlo-labs/metlo/ingestors/govxlan/utils"
	"github.com/sirupsen/logrus"
)

type Metlo struct {
	mu        sync.Mutex
	ts        []int64
	rps       int
	metloHost string
	metloKey  string
}

const MetloDefaultRPS int = 10

func InitMetlo(metloHost string, metloKey string, rps int) *Metlo {
	inst := &Metlo{
		ts:        make([]int64, 0, rps),
		rps:       rps,
		metloHost: metloHost + "/api/v1/log-request/single",
		metloKey:  metloKey,
	}
	return inst
}

func (m *Metlo) Send(data MetloTrace) {
	traceHost := data.Request.Url.Host
	httpTraceHost := fmt.Sprintf("http://%s", traceHost)
	httpsTraceHost := fmt.Sprintf("https://%s", traceHost)
	if strings.HasPrefix(m.metloHost, httpTraceHost) || strings.HasPrefix(m.metloHost, httpsTraceHost) {
		utils.Log.Trace("Skipped Request to Metlo Host.")
		return
	}
	utils.Log.WithFields(logrus.Fields{
		"Method": data.Request.Method,
		"Host":   data.Request.Url.Host,
		"Path":   data.Request.Url.Path,
	}).Trace("Sending Request.")
	json, err := json.Marshal(data)
	if err != nil {
		utils.Log.WithError(err).Debug("Error Building Request.")
		return
	}
	req, err := http.NewRequest("POST", m.metloHost, bytes.NewBuffer(json))
	if err != nil {
		utils.Log.WithError(err).Debug("Error Building Request.")
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", m.metloKey)
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		utils.Log.WithError(err).Debug("Error Sending Request.")
	}
	if resp == nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		message, _ := io.ReadAll(resp.Body)
		utils.Log.WithFields(logrus.Fields{
			"Method":  data.Request.Method,
			"Host":    data.Request.Url.Host,
			"Path":    data.Request.Url.Path,
			"Code":    resp.Status,
			"Message": string(message),
		}).Debug("Error Sending Request.")
	} else {
		utils.Log.WithFields(logrus.Fields{
			"Method": data.Request.Method,
			"Host":   data.Request.Url.Host,
			"Path":   data.Request.Url.Path,
		}).Trace("Sent Request.")
	}
}

func (m *Metlo) Allow() bool {
	m.mu.Lock()
	tmp_ts := make([]int64, 0, 10)
	now := time.Now()
	curr := now.UTC().UnixMilli()
	if len(m.ts) == 0 {

	} else {
		for x := 0; x < len(m.ts); x++ {
			if (curr - m.ts[x]) <= 1000 {
				tmp_ts = append(tmp_ts, m.ts[x])
			}
		}
	}
	m.ts = tmp_ts
	if len(m.ts) < m.rps {
		m.ts = append(m.ts, curr)
		m.mu.Unlock()
		return true
	}
	m.mu.Unlock()
	return false
}
