package main

import (
	"encoding/json"

	"github.com/rs/zerolog/log"
)

func init() {
	RegisterHandler("User.fullName2", func(rq Req) (result []byte) {
		res := make([]string, 0)
		for _, v := range rq.Parents {
			res = append(res, "Golang: "+v["firstName"].(string)+" "+v["lastName"].(string))
		}
		var err error
		result, err = json.Marshal(res)
		if err != nil {
			log.Fatal().Err(err).Msg("Cannot marshall")
		}
		return
	})
}
