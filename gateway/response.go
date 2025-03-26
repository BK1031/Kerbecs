package gateway

import (
	"encoding/json"
	"io"
	"kerbecs/config"
	"kerbecs/model"
	"kerbecs/utils"
	"net/http"

	"github.com/bk1031/rincon-go/v2"
)

func BuildResponseStruct(response *http.Response, proxiedService rincon.Service) (model.Response, error) {
	respModel := model.Response{
		Gateway: config.Service.FormattedNameWithVersion(),
		Service: proxiedService.FormattedNameWithVersion(),
	}
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		utils.SugarLogger.Errorln("Failed to read response body: " + err.Error())
		return respModel, err
	}
	err = json.Unmarshal(bodyBytes, &respModel.Data)
	if err != nil {
		utils.SugarLogger.Errorln("Failed to unmarshall response body, returning as message string: " + err.Error())
		respModel.Data = json.RawMessage("{\"message\": \"" + string(bodyBytes) + "\"}")
	}
	if response.StatusCode < 200 {
		respModel.Status = "INFO"
	} else if response.StatusCode < 300 {
		respModel.Status = "SUCCESS"
	} else if response.StatusCode < 400 {
		respModel.Status = "REDIRECT"
	} else {
		respModel.Status = "ERROR"
	}
	return respModel, nil
}
