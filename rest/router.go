package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Kaese72/sdup-lib/httpsdup"
	"github.com/Kaese72/sdup-lib/sduptemplates"
	"github.com/Kaese72/sdup-lib/subscription"
	"github.com/Kaese72/sdup-rest/cache"
	"github.com/gorilla/mux"
)

//InitSDUPCachedRest initializes a HTTP server mux with the appropriate paths
func InitSDUPCachedRest(cache cache.SDUPCache) *mux.Router {
	_, channel, err := cache.Initialize()
	if err != nil {
		//FIXME No reason to panic
		panic(err)
	}
	subs := subscription.NewSubscriptions(channel)
	router := mux.NewRouter()
	router.HandleFunc("/discovery", func(writer http.ResponseWriter, reader *http.Request) {
		devices, err := cache.Devices()
		if err != nil {
			//log.Log(log.Error, err.Error(), nil)
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonEncoded, err := json.MarshalIndent(devices, "", "   ")
		if err != nil {
			//log.Log(log.Error, err.Error(), nil)
			http.Error(writer, "Failed to JSON encode SDUPDevices", http.StatusInternalServerError)
		}
		writer.Write(jsonEncoded)
	})
	router.HandleFunc("/subscribe", func(writer http.ResponseWriter, reader *http.Request) {
		//log.Log(log.Info, "Started SSE handler", nil)
		// prepare the header
		writer.Header().Set("Content-Type", "text/event-stream")
		writer.Header().Set("Cache-Control", "no-cache")
		writer.Header().Set("Connection", "keep-alive")
		writer.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, _ := writer.(http.Flusher)

		subscription := subs.Subscribe()
		doneChan := reader.Context().Done()
		for {

			select {
			// connection is closed then defer will be executed
			case <-doneChan:
				// Communicate the cancellation of this subscription
				subs.UnSubscribe(subscription)
				doneChan = nil

			case event, ok := <-subscription.Updates():
				if ok {
					jsonString, err := json.Marshal(event)
					if err != nil {
						//log.Log(log.Error, "Failed to Marshal device update", nil)

					} else {
						fmt.Fprintf(writer, "data: %s\n\n", jsonString)
						flusher.Flush()
					}

				} else {
					return
				}
			}
		}
	})

	router.HandleFunc("/capability/{deviceID}/{capabilityKey}", func(writer http.ResponseWriter, reader *http.Request) {
		vars := mux.Vars(reader)
		deviceID := vars["deviceID"]
		capabilityKey := vars["capabilityKey"]
		//log.Log(log.Info, "Triggering capability", map[string]string{"device": deviceID, "capability": capabilityKey})
		var args sduptemplates.CapabilityArgument

		err := json.NewDecoder(reader.Body).Decode(&args)
		if err != nil {
			if err == io.EOF {
				//No body was sent. That is fine
				args = sduptemplates.CapabilityArgument{}
			} else {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
		}
		err = cache.TriggerCapability(sduptemplates.DeviceID(deviceID), sduptemplates.CapabilityKey(capabilityKey), args)
		if err != nil {
			//FIXME Do not use httpsdup
			http.Error(writer, err.Error(), httpsdup.HTTPStatusCode(err))
			return

		}
		http.Error(writer, "OK", http.StatusOK)

	}).Methods("POST")
	//router.PathPrefix("/ui/").Handler(http.StripPrefix("/ui/", http.FileServer(http.Dir("./ui/"))))
	return router
}
