package controllers

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/otiai10/gosseract/v2"
	"github.com/otiai10/marmoset"
)

var (
	imgexp = regexp.MustCompile("^image")
)

type OcrRes struct {
	Word       string
	Confidence float64
}

// FileUpload ...
func FileUpload(w http.ResponseWriter, r *http.Request) {

	render := marmoset.Render(w, true)

	// Get uploaded file
	r.ParseMultipartForm(32 << 20)
	// upload, h, err := r.FormFile("file")
	upload, _, err := r.FormFile("file")
	if err != nil {
		render.JSON(http.StatusBadRequest, err)
		return
	}
	defer upload.Close()

	// Create physical file
	tempfile, err := ioutil.TempFile("", "ocrserver"+"-")
	if err != nil {
		render.JSON(http.StatusBadRequest, err)
		return
	}
	defer func() {
		tempfile.Close()
		os.Remove(tempfile.Name())
	}()

	// Make uploaded physical
	if _, err = io.Copy(tempfile, upload); err != nil {
		render.JSON(http.StatusInternalServerError, err)
		return
	}

	client := gosseract.NewClient()
	defer client.Close()

	client.SetImage(tempfile.Name())
	client.Languages = []string{"eng"}
	if langs := r.FormValue("languages"); langs != "" {
		client.Languages = strings.Split(langs, ",")
	}
	if whitelist := r.FormValue("whitelist"); whitelist != "" {
		client.SetWhitelist(whitelist)
	}

	var out string
	switch r.FormValue("format") {
	case "hocr":
		out, err = client.HOCRText()
		render.EscapeHTML = false
	case "json":
		var out1 []gosseract.BoundingBox
		var out2 []byte
		var out3 []OcrRes
		out1, err = client.GetBoundingBoxes(gosseract.RIL_WORD)
		if err != nil {
			render.JSON(http.StatusBadRequest, err)
			return
		}
		for _, x := range out1 {
			ir := OcrRes{Word: x.Word, Confidence: x.Confidence}
			out3 = append(out3, ir)
		}
		out2, err = json.Marshal(out3)
		out = string(out2)
	default:
		out, err = client.Text()
	}
	if err != nil {
		render.JSON(http.StatusBadRequest, err)
		return
	}

	render.JSON(http.StatusOK, map[string]interface{}{
		"result":  strings.Trim(out, r.FormValue("trim")),
		"version": version,
	})
}
