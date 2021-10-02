package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Kaese72/sdup-lib/httpsdup"
	"github.com/Kaese72/sdup-lib/logging"
	"github.com/Kaese72/sdup-lib/sduptemplates"
	"github.com/Kaese72/sdup-lib/subscription"
	"github.com/Kaese72/sdup-rest/cache"
	"github.com/Kaese72/sdup-rest/cache/filters"
	"github.com/gorilla/mux"
)

type SDUPRest struct {
	authentication JWTWrapper
	config         httpsdup.Config
	cache          cache.SDUPCache
}

func NewSDUPRestCache(config httpsdup.Config, cache cache.SDUPCache) *SDUPRest {
	var rest SDUPRest
	rest.config = config
	//FIXME Config for key
	rest.authentication = NewJWTWrapper("kindofsecretkey", "sdup-rest", 5, 24)
	rest.cache = cache

	return &rest
}

const cookieName = "sdup-refresh-1"
const loginPath = "/rest/v0/auth/login"

func (rest *SDUPRest) ListenAndServe() error {
	_, channel, err := rest.cache.Initialize()
	if err != nil {
		//FIXME No reason to panic
		panic(err)
	}
	subs := subscription.NewSubscriptions(channel)
	router := mux.NewRouter()

	router.HandleFunc(loginPath, func(writer http.ResponseWriter, reader *http.Request) {
		// ############################################
		// # Try to authenticate with cookie contents #
		// ############################################
		//FIXME What about multiple cookies ?
		cookie, err := reader.Cookie(cookieName)
		if err == nil {
			if !cookie.Expires.After(time.Now()) {
				//Cookie is not expired and can be used
				user, err := rest.authentication.ValidateToken(cookie.Value)
				if err != nil {
					http.Error(writer, "Could not parse cookie content", http.StatusInternalServerError)
					return
				}
				token, err := rest.authentication.GenerateLoginToken(user.Name)
				if err != nil {
					http.Error(writer, err.Error(), http.StatusForbidden)
					return
				}
				http.Error(writer, fmt.Sprintf("{\"token\": \"%s\"}", token), http.StatusOK)
				return
			}
			//Cookie is expired. Move along
		}

		// ##########################################
		// # Try to authenticate with body contents #
		// ##########################################
		var login LoginBody
		err = json.NewDecoder(reader.Body).Decode(&login)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		token, err := rest.authentication.UserPassToToken(login.User, login.Password)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusForbidden)
			return
		}

		refreshToken, err := rest.authentication.GenerateRefreshToken(login.User)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
		refreshCookie := http.Cookie{Name: cookieName, Value: refreshToken, HttpOnly: true, Path: loginPath}
		http.SetCookie(writer, &refreshCookie)
		http.Error(writer, fmt.Sprintf("{\"token\": \"%s\"}", token), http.StatusOK)
	}).Methods("POST")

	//Everything else (not /auth/login) should have the authentication middleware
	apiv0 := router.PathPrefix("/rest/v0/").Subrouter()
	apiv0.Use(rest.authenticationMiddleware)

	apiv0.HandleFunc("/devices", func(writer http.ResponseWriter, reader *http.Request) {
		attrFilters := filters.AttributeFilters{}
		if afparams, ok := reader.URL.Query()["attributefilter"]; ok {
			for _, afparam := range afparams {
				var ps filters.AttributeFilters
				err := json.Unmarshal([]byte(afparam), &ps)
				if err != nil {
					http.Error(writer, err.Error(), http.StatusInternalServerError)
					return
				}
				attrFilters = append(attrFilters, ps...)
			}
		}

		devices, err := rest.cache.Devices(attrFilters)
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

	apiv0.HandleFunc("/devices/{deviceID}", func(writer http.ResponseWriter, reader *http.Request) {
		vars := mux.Vars(reader)
		deviceID := vars["deviceID"]

		device, err := rest.cache.Device(sduptemplates.DeviceID(deviceID))
		if err != nil {
			cache.ServeErrorContent(err, writer)
		}
		jsonEncoded, err := json.MarshalIndent(device, "", "   ")
		if err != nil {
			//log.Log(log.Error, err.Error(), nil)
			http.Error(writer, "Failed to JSON encode SDUPDevices", http.StatusInternalServerError)
		}
		writer.Write(jsonEncoded)

	}).Methods("GET")

	apiv0.HandleFunc("/capability/{deviceID}/{capabilityKey}", func(writer http.ResponseWriter, reader *http.Request) {
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
		err = rest.cache.TriggerCapability(sduptemplates.DeviceID(deviceID), sduptemplates.CapabilityKey(capabilityKey), args)
		if err != nil {
			//FIXME Do not use httpsdup
			http.Error(writer, err.Error(), httpsdup.HTTPStatusCode(err))
			return

		}
		http.Error(writer, "OK", http.StatusOK)

	}).Methods("POST")

	apiv0.HandleFunc("/subscribe", func(writer http.ResponseWriter, reader *http.Request) {
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

	if err := http.ListenAndServe(fmt.Sprintf("%s:%d", rest.config.ListenAddress, rest.config.ListenPort), router); err != nil {
		logging.Error(err.Error())
		return err
	}
	return nil
}

func (rest *SDUPRest) authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, reader *http.Request) {
		authHeaderVal := reader.Header.Get("authorization")
		// FIXME improve bearer parsing. bearer should be case insensitive
		if authHeaderVal == "" || !strings.HasPrefix(strings.ToLower(authHeaderVal), "bearer ") {
			http.Error(writer, "You are not logged in", http.StatusForbidden)
			return
		}
		token := authHeaderVal[7:]
		_, err := rest.authentication.ValidateToken(token)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusForbidden)
			return
		}
		next.ServeHTTP(writer, reader)
	})
}
