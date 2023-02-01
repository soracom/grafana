package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/org"
)

var ImageLinkerUrl = os.Getenv("IMAGE_LINKER_URL")
var ImageLinkerKey = os.Getenv("IMAGE_LINKER_KEY")

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func generateSignature(key string, signType string, orgName string, signTime string) string {

	// String to sign is something like "LagoonLogo:My Org:156465484894"
	signature := hmacSHA256([]byte(key), []byte(signType+":"+orgName+":"+signTime))
	// Convert the signature into something readable
	signatureStr := fmt.Sprintf("%x", signature)

	return signatureStr

}

// GetLogoUploadLink GET /api/orgs/images/link
func (hs *HTTPServer) GetLogoUploadLink(c *models.ReqContext) response.Response {

	query := org.GetOrgByIdQuery{ID: c.OrgID}

	res, err := hs.orgService.GetByID(c.Req.Context(), &query)
	if err != nil {
		if errors.Is(err, models.ErrOrgNotFound) {
			return response.Error(http.StatusNotFound, "Organization not found", err)
		}
		return response.Error(http.StatusInternalServerError, "Failed to get organization", err)
	}

	orga := res

	linkerURL := ImageLinkerUrl
	signingKey := ImageLinkerKey
	signatureType := "LagoonLogo"

	if len(linkerURL) == 0 {
		return response.Error(500, "Image Linker URL error", errors.New("setting.ImageLinkerUrl is empty"))
	}
	if len(signingKey) == 0 {
		return response.Error(500, "Image Linker Signing error", errors.New("setting.ImageLinkerKey is empty"))
	}

	orgName := orga.Name

	if !strings.HasSuffix(orgName, "-PRO") {
		return response.Error(500, "api.authenticationError", errors.New("Organisation doesn't have PRO suffix"))
	}

	signTime := fmt.Sprintf("%v", time.Now().Unix())
	signatureStr := generateSignature(signingKey, signatureType, orgName, signTime)

	req, _ := http.NewRequest("GET", linkerURL, nil)

	q := req.URL.Query()
	q.Add("type", signatureType)
	q.Add("auth", signatureStr)
	q.Add("id", orgName)
	q.Add("time", signTime)
	req.URL.RawQuery = q.Encode()

	fmt.Println(req.URL.String())

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return response.Error(500, "api.authenticationError", err)
	}

	if resp.StatusCode != 200 {
		return response.Error(500, "org.organizationNotFound", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return response.Error(500, "api.authenticationError", err)
	}

	return response.JSON(200, body)
}
