package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"time"
)

var (
	modemUri      = EnvStringReq("MODEM_URI")
	modemUsername = EnvStringReq("MODEM_USERNAME")
	modemPassword = EnvStringReq("MODEM_PASSWORD")

	influxUri      = EnvStringReq("INFLUXDB_URI")
	influxToken    = EnvStringReq("INFLUXDB_TOKEN")
	influxBucket   = EnvStringReq("INFLUXDB_BUCKET_ID")
	influxOrg      = EnvStringReq("INFLUXDB_ORG_ID")
	influxLocation = EnvString("INFLUXDB_TAG", "pk5001z")
)

func main() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)

	loop()

	t := time.NewTicker(time.Minute)
	for {
		select {
		case <-t.C:
			loop()
		case <-signalChannel:
			return
		}
	}
}

func loop() {
	options := cookiejar.Options{}
	jar, err := cookiejar.New(&options)
	if err != nil {
		log.Fatal(err)
	}
	client := http.Client{Jar: jar}

	/* Authenticate */
	loginData := url.Values{
		"loginSubmitValue": {"1"},
		"admin_username":   {modemUsername},
		"admin_password":   {modemPassword},
	}
	resp, err := client.PostForm(modemUri+"/login.cgi", loginData)
	if err != nil {
		log.Printf("Error : %s\n", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Error : %s\n", resp.Status)
		log.Println(string(body))
		return
	}

	/* Get Data */
	resp, err = client.Get(modemUri + "/GetDSLInfo.cgi")
	if err != nil {
		log.Printf("Error : %s\n", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	preSplit := strings.ReplaceAll(string(body), "||", "|")
	parts := strings.Split(preSplit, "|")
	for i, v := range parts {
		parts[i] = strings.TrimSpace(v)
	}
	uptime, err := time.ParseDuration(strings.ToLower(strings.ReplaceAll(parts[6], ":", "")))
	if err != nil {
		uptime = time.Duration(0)
	}

	stats := DslStats{
		Down:       parts[2],
		Up:         parts[3],
		LinkUptime: fmt.Sprintf("%d", uptime.Milliseconds()/1000),
		Retrains:   parts[7],
		//LossOfPowerLink:    parts[16],
		//LossOfSignalLink:   parts[18],
		//LinkTrainErrors:    parts[20],
		UnavailableSeconds: parts[13],
		SNRDown:            parts[19],
		SNRUp:              parts[20],
		AttenuationDown:    parts[21],
		AttenuationUp:      parts[22],
		PowerDown:          parts[23],
		PowerUp:            parts[24],
		PacketsDown:        parts[25],
		PacketsUp:          parts[26],
		ErrorPacketsDown:   parts[27],
		ErrorPacketsUp:     parts[28],
		CRCNearEnd:         parts[37],
		CRCFarEnd:          parts[38],
		RSNearEnd:          parts[41],
		RSFarEnd:           parts[42],
	}

	write(stats)
}

func write(stats DslStats) {
	fields := make([]string, 0, 20)

	val := reflect.ValueOf(stats)
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		if valueField.Interface() == "" {
			continue
		}
		fields = append(fields, fmt.Sprintf("%s=%v", typeField.Name, valueField.Interface()))
	}
	if len(fields) == 0 {
		log.Println("No fields to send...")
		return
	}
	fieldsString := strings.Join(fields, ",")
	line := fmt.Sprintf("dsl,location=%s %s %d", influxLocation, fieldsString, time.Now().UnixNano())

	client := http.Client{}
	endpoint := fmt.Sprintf("%s/api/v2/write?org=%s&bucket=%s&precision=ns", influxUri, influxOrg, influxBucket)
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(line))
	if err != nil {
		log.Println(err)
		return
	}
	req.Header.Add("Authorization", fmt.Sprintf("Token %s", influxToken))
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	if resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Error : %s\n", resp.Status)
		log.Println(string(body))
		return
	}
	resp.Body.Close()

	log.Println(line)
}

func EnvString(key string, _default string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return _default
	}
	return val
}

func EnvStringReq(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("required env variable not set: %s", key))
	}
	return val
}
