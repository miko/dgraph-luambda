package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/seatgeek/graphql"

	"github.com/rs/zerolog/log"
)

type graphqlcfg struct {
	SchemaFile      string //`default:"schema.graphql"`
	DataFile        string //`default:"init.graphql"`
	SchemaEndPoint  string `default:"http://dgraph:8080/admin/schema"`
	AlterEndPoint   string `default:"http://dgraph:8080/alter"`
	GraphqlEndPoint string `default:"http://dgraph:8080/graphql"`
	DropAll         bool   `default:"false"`
}

func init() {
	var g graphqlcfg
	envconfig.Process("dgraph", &g)
	if g.DropAll {
		err := dropData(g.AlterEndPoint)
		if err != nil {
			log.Error().Err(err).Str("endpoint", g.AlterEndPoint).Msg("Cannot drop data")
		} else {
			log.Info().Str("endpoint", g.AlterEndPoint).Msg("Dropped data")
		}
	} else {
		log.Debug().Msg("Not dropping data")
	}
	if g.SchemaFile != "" {
		err := uploadSchema(g.SchemaEndPoint, g.SchemaFile)
		if err != nil {
			log.Error().Err(err).Str("schemafile", g.SchemaFile).Str("endpoint", g.SchemaEndPoint).Msg("Cannot upload schema")
		} else {
			log.Info().Str("schemafile", g.SchemaFile).Str("endpoint", g.SchemaEndPoint).Msg("Uploaded schema")
		}
	}
	if g.DataFile != "" {
		err := uploadGraphql(g.GraphqlEndPoint, g.DataFile)
		if err != nil {
			log.Error().Err(err).Str("datafile", g.DataFile).Str("endpoint", g.GraphqlEndPoint).Msg("Cannot upload data")
		} else {
			log.Info().Str("datafile", g.DataFile).Str("endpoint", g.GraphqlEndPoint).Msg("Uploaded data")
		}
	}
}

type schemaResponse struct {
	Errors []struct {
		Message string `json:"message,omitempty"`
	} `json:"errors,omitempty"`
	Data struct {
		Message string `json:"message,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"data,omitempty"`
}

func dropData(endpoint string) error {
	count := 0
	done := false
	log.Warn().Msg("Dropping all data")
	for {
		count++
		time.Sleep(time.Second)
		resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer([]byte(`{"drop_all": true}`)))
		if err != nil {
			log.Error().Err(err).Int("step", count).Msg("Cannot drop data")
			continue
		} else {
			log.Info().Int("step", count).Msgf("Got resp %#v", resp)
			if resp.StatusCode == 200 {
				done = true
				break
			} else {
				log.Error().Err(err).Int("step", count).Int("code", resp.StatusCode).Msg("Cannot drop data")
			}
		}
	}
	if done {
		return nil
	} else {
		return errors.New("Cannot drop data")
	}
}

func uploadSchema(endpoint, filename string) error {
	count := 0
	//	f, err := ioutil.ReadFile(filename)
	done := false
	log.Info().Str("filename", filename).Str("endpoint", endpoint).Msg("Uploading schema")
	for {
		count++
		time.Sleep(time.Second)
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		resp, err := http.Post(endpoint, "application/json", f)
		if err != nil {
			//return err
			log.Error().Err(err).Int("step", count).Msg("Cannot post schema")
			continue
		}
		buf, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Error().Err(err).Int("step", count).Msg("Cannot read response")
			//			return err
			continue
		}
		log.Info().Str("response", string(buf)).Int("step", count).Msg("Uploaded schema")
		var r schemaResponse
		err = json.Unmarshal(buf, &r)
		if err != nil {
			log.Error().Err(err).Int("step", count).Msg("Cannot unmarshall response")
			//			return err
			continue
		}
		if len(r.Errors) == 0 && r.Data.Code == "Success" {
			done = true
			break
		} else {
			log.Info().Str("error", r.Errors[0].Message).Int("step", count).Msg("Did not upload schema")
		}
	}
	if done {
		return nil
	} else {
		return errors.New("Cannot upload schema")
	}
}
func uploadGraphql(endpoint, filename string) error {
	count := 0
	done := false
	log.Info().Str("filename", filename).Str("endpoint", endpoint).Msg("Uploading data")
	for {
		count++
		time.Sleep(time.Second)
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		f.Close()
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		client := graphql.NewClient(endpoint)

		// make a request
		req := graphql.NewRequest(string(b))
		req.Header.Set("Cache-Control", "no-cache")
		ctx := context.Background()
		var respData interface{}
		if err := client.Run(ctx, req, &respData); err != nil {
			//log.Fatal(err)
			log.Error().Err(err).Int("step", count).Msg("Cannot post data")
			continue
		}

		log.Info().Int("step", count).Msgf("Uploaded data, resp: %#v", respData)
		done = true
		break
		/*
			if len(r.Errors) == 0 && r.Data.Code == "Success" {
				done = true
				break
			} else {
				log.Info().Str("error", r.Errors[0].Message).Int("step", count).Msg("Did not upload schema")
			}
		*/
	}
	if done {
		return nil
	} else {
		return errors.New("Cannot upload data")
	}
}
