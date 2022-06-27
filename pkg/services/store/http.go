package store

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/web"
)

var errFileTooBig = response.Error(400, "Please limit file uploaded under 1MB", errors.New("file is too big"))

// HTTPStorageService passes raw HTTP requests to a well typed storage service
type HTTPStorageService interface {
	List(c *models.ReqContext) response.Response
	Read(c *models.ReqContext) response.Response
	Delete(c *models.ReqContext) response.Response
	Upload(c *models.ReqContext) response.Response
}

type httpStorage struct {
	store StorageService
}

func ProvideHTTPService(store StorageService) HTTPStorageService {
	return &httpStorage{
		store: store,
	}
}

func UploadErrorToStatusCode(err error) int {
	switch {
	case errors.Is(err, ErrUploadFeatureDisabled):
		return 404

	case errors.Is(err, ErrUnsupportedStorage):
		return 400

	case errors.Is(err, ErrValidationFailed):
		return 400

	case errors.Is(err, ErrFileAlreadyExists):
		return 400

	default:
		return 500
	}
}

func (s *httpStorage) Upload(c *models.ReqContext) response.Response {
	// 32 MB is the default used by FormFile()
	if err := c.Req.ParseMultipartForm(32 << 20); err != nil {
		return response.Error(400, "error in parsing form", err)
	}
	c.Req.Body = http.MaxBytesReader(c.Resp, c.Req.Body, MAX_UPLOAD_SIZE)
	if err := c.Req.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		return response.Error(400, "Please limit file uploaded under 1MB", err)
	}

	files := c.Req.MultipartForm.File["file"]
	if len(files) != 1 {
		return response.JSON(400, map[string]interface{}{
			"message": "please upload files one at a time",
			"err":     true,
		})
	}

	fileHeader := files[0]
	if fileHeader.Size > MAX_UPLOAD_SIZE {
		return errFileTooBig
	}

	// restrict file size based on file size
	// open each file to copy contents
	file, err := fileHeader.Open()
	if err != nil {
		return response.Error(500, "Internal Server Error", err)
	}
	err = file.Close()
	if err != nil {
		return response.Error(500, "Internal Server Error", err)
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return response.Error(500, "Internal Server Error", err)
	}

	if (len(data)) > MAX_UPLOAD_SIZE {
		return errFileTooBig
	}

	path := RootUpload + "/" + fileHeader.Filename

	mimeType := http.DetectContentType(data)

	err = s.store.Upload(c.Req.Context(), c.SignedInUser, &UploadRequest{
		Contents:              data,
		MimeType:              mimeType,
		EntityType:            EntityTypeImage,
		Path:                  path,
		OverwriteExistingFile: true,
	})

	if err != nil {
		return response.Error(UploadErrorToStatusCode(err), err.Error(), err)
	}

	return response.JSON(200, map[string]interface{}{
		"message": "Uploaded successfully",
		"path":    path,
		"file":    fileHeader.Filename,
		"err":     true,
	})
}

func (s *httpStorage) Read(c *models.ReqContext) response.Response {
	// full path is api/storage/read/upload/example.jpg, but we only want the part after read
	scope, path := getPathAndScope(c)
	file, err := s.store.Read(c.Req.Context(), c.SignedInUser, scope+"/"+path)
	if err != nil {
		return response.Error(400, "cannot call read", err)
	}
	// set the correct content type for svg
	if strings.HasSuffix(path, ".svg") {
		c.Resp.Header().Set("Content-Type", "image/svg+xml")
	}
	return response.Respond(200, file.Contents)
}

func (s *httpStorage) Delete(c *models.ReqContext) response.Response {
	// full path is api/storage/delete/upload/example.jpg, but we only want the part after upload
	_, path := getPathAndScope(c)
	err := s.store.Delete(c.Req.Context(), c.SignedInUser, "/"+path)
	if err != nil {
		return response.Error(400, "cannot call delete", err)
	}
	return response.JSON(200, map[string]string{
		"message": "Removed file from storage",
		"path":    path,
	})
}

func (s *httpStorage) List(c *models.ReqContext) response.Response {
	params := web.Params(c.Req)
	path := params["*"]
	frame, err := s.store.List(c.Req.Context(), c.SignedInUser, path)
	if err != nil {
		return response.Error(400, "error reading path", err)
	}
	if frame == nil {
		return response.Error(404, "not found", nil)
	}
	return response.JSONStreaming(http.StatusOK, frame)
}
