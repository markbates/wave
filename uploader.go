package wave

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	humanize "github.com/dustin/go-humanize"
	"github.com/markbates/validate"
	"github.com/markbates/validate/validators"
	"github.com/pkg/errors"
)

type FileTypeValidator struct {
	Field        string
	AllowedTypes map[string]bool
	Headers      *multipart.FileHeader
}

func (f FileTypeValidator) IsValid(errors *validate.Errors) {
	key := validators.GenerateKey(f.Field)
	if !f.AllowedTypes[f.Headers.Header.Get("Content-Type")] {
		errors.Add(key, "not an allowed type")
	}
}

type MaxFileSizeValidator struct {
	Field   string
	MaxSize int
	Headers *multipart.FileHeader
}

func (m MaxFileSizeValidator) IsValid(errors *validate.Errors) {
	key := validators.GenerateKey(m.Field)
	s, err := strconv.Atoi(m.Headers.Header.Get("Content-Length"))
	if err != nil {
		errors.Add(key, "couldn't parse content length")
		return
	}
	if s > m.MaxSize {
		errors.Add(key, fmt.Sprintf("is too big %s", humanize.Bytes(uint64(s))))
	}
}

type Uploader interface {
	FieldName() string
	Path(*multipart.FileHeader) string
	Validate(*multipart.FileHeader) (*validate.Errors, error)
	Put(path string, in io.Reader, size int64, contentType string) error
}

func Upload(req *http.Request, u Uploader) (*validate.Errors, error) {
	verrs := validate.NewErrors()
	f, h, err := req.FormFile(u.FieldName())
	if err != nil {
		if err == http.ErrMissingFile {
			return verrs, nil
		}
		return verrs, errors.WithStack(err)
	}
	var fSize int64
	if h.Header.Get("Content-Length") == "" {
		fSize, err = size(f)
		if err != nil {
			return verrs, errors.WithStack(err)
		}
		h.Header.Set("Content-Length", fmt.Sprint(fSize))
	} else {
		s, err := strconv.Atoi(h.Header.Get("Content-Length"))
		if err != nil {
			return verrs, errors.WithStack(err)
		}
		fSize = int64(s)
	}
	verrs, err = u.Validate(h)
	if err != nil {
		return verrs, errors.WithStack(err)
	}
	if verrs.HasAny() {
		return verrs, nil
	}
	err = u.Put(u.Path(h), f, fSize, h.Header.Get("Content-Type"))
	if err != nil {
		return verrs, errors.WithStack(err)
	}
	return verrs, nil
}

func size(f multipart.File) (int64, error) {
	size, err := f.Seek(0, 2) //2 = from end
	if err != nil {
		return 0, errors.WithStack(err)
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return size, nil
}
