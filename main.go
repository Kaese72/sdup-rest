package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Kaese72/sdup-lib/logging"
	"github.com/Kaese72/sdup-lib/sdupclient"
	"github.com/Kaese72/sdup-rest/cache"
	"github.com/Kaese72/sdup-rest/config"
	"github.com/Kaese72/sdup-rest/rest"
)

func main() {
	var conf config.Config

	if _, err := os.Stat("./settings.json"); err == nil {
		file, err := os.Open("./settings.json")
		if err != nil {
			logging.Error(fmt.Sprintf("Unable to open local settings file, %s", err.Error()))
			return
		}
		if err := json.NewDecoder(file).Decode(&conf); err != nil {
			logging.Error(err.Error())
		}

	} else {
		if err := json.NewDecoder(os.Stdin).Decode(&conf); err != nil {
			logging.Error(err.Error())
		}
	}

	if err := conf.Validate(); err != nil {
		logging.Error(err.Error())
		conf.PopulateExample()
		obj, err := json.Marshal(conf)
		if err != nil {
			logging.Error(err.Error())
		}
		_, err = fmt.Fprintf(os.Stdout, "%s\n", obj)
		if err != nil {
			logging.Error(err.Error())
		}
		return
	}

	sdupClient, err := sdupclient.NewSDUPClient(conf.SDUPClientConfig)
	if err != nil {
		logging.Error(err.Error())
		return
	}
	sdupCache := cache.NewSDUPCache(sdupClient)
	router := rest.NewSDUPRestCache(conf.SDUPServerConfig, sdupCache)
	router.ListenAndServe()
}
