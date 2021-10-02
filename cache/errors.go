package cache

import (
	"fmt"
	"net/http"

	"github.com/Kaese72/sdup-rest/faults"
)

func ServeErrorContent(err error, writer http.ResponseWriter) {
	switch err.(type) {
	case faults.ErrEntityNotFound:
		http.Error(writer, fmt.Sprintf("Not found: %s", err.Error()), http.StatusNotFound)

	default:
		http.Error(writer, fmt.Sprintf("Unknown error: %s", err.Error()), http.StatusInternalServerError)
	}
}
