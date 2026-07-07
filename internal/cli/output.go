package cli

import (
	"encoding/json"
	"fmt"
	"io"

	toon "github.com/toon-format/toon-go"
)

func writeOutput(w io.Writer, value any, asJSON bool) error {
	var data []byte
	var err error
	if asJSON {
		data, err = json.MarshalIndent(value, "", "  ")
	} else {
		data, err = toon.Marshal(value, toon.WithLengthMarkers(true))
	}
	if err != nil {
		return err
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	_, err = fmt.Fprint(w, string(data))
	return err
}
