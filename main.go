package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Kaese72/sdup-lib/logging"
	"github.com/Kaese72/sdup-lib/sdupclient"
	"github.com/Kaese72/sdup-rest/cache"
	"github.com/Kaese72/sdup-rest/rest"
)

func main() {
	var conf Config

	if err := json.NewDecoder(os.Stdin).Decode(&conf); err != nil {
		logging.Error(err.Error())
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
	router := rest.InitSDUPCachedRest(sdupCache)
	if err := http.ListenAndServe(fmt.Sprintf("%s:%d", conf.SDUPServerConfig.ListenAddress, conf.SDUPServerConfig.ListenPort), router); err != nil {
		logging.Error(err.Error())
		return
	}
}
