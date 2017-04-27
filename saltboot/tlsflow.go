package saltboot

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/hortonworks/salt-bootstrap/saltboot/cautils"
	"github.com/hortonworks/salt-bootstrap/saltboot/model"
)

type Credentials struct {
	Clients
	PublicIP  *string `json:"PublicIP" yaml:"PublicIP"`
	AuthToken *string `json:"AuthToken" yaml:"AuthToken"`
}

func ClientCredsHandler(w http.ResponseWriter, req *http.Request) {

	decoder := json.NewDecoder(req.Body)
	var credentials Credentials
	err := decoder.Decode(&credentials)
	if err != nil {
		log.Printf("[ClientCredsHandler] [ERROR] couldn't decode json: %s", err.Error())
		model.Response{Status: err.Error()}.WriteBadRequestHttp(w)
		return
	}

	// mkdir if needed
	log.Printf("[CAHandler] handleClientCreds executed")
	w.Header().Set("Content-Type", "application/json")
	pubIp := credentials.PublicIP
	authToken := credentials.AuthToken
	if cautils.IsPathExisting(cautils.DetermineCrtDir(os.Getenv)) == false {
		if err := os.Mkdir(cautils.DetermineCrtDir(os.Getenv), 0755); err != nil {
			log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
			model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
			return
		}
	}
	caResp, _ := http.Get("http://" + credentials.Servers[0].Address + ":7070/saltboot/ca")
	caBytes, _ := ioutil.ReadAll(caResp.Body)
	caCrt, err := cautils.NewCertificateFromPEM(caBytes)
	if err != nil {
		log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
		model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
		return
	}
	err = caCrt.ToPEMFile(filepath.Join(cautils.DetermineCrtDir(os.Getenv), "ca.crt"))
	if cautils.IsPathExisting(filepath.Join(cautils.DetermineCrtDir(os.Getenv), "client.key")) == false {
		key, err := cautils.NewKey()
		if err != nil {
			log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
			model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
			return
		}

		err = key.ToPEMFile(filepath.Join(cautils.DetermineCrtDir(os.Getenv), "client.key"))
		if err != nil {
			log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
			model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
			return
		}
	}
	if cautils.IsPathExisting(filepath.Join(cautils.DetermineCrtDir(os.Getenv), "client.csr")) == false {
		key, err := cautils.NewKeyFromPrivateKeyPEMFile(filepath.Join(cautils.DetermineCrtDir(os.Getenv), "client.key"))
		if err != nil {
			log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
			model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
			return
		}

		csr, err := cautils.NewCertificateRequest(key, pubIp)
		if err != nil {
			log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
			model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
			return
		}
		err = csr.ToPEMFile(filepath.Join(cautils.DetermineCrtDir(os.Getenv), "client.csr"))
		if err != nil {
			log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
			model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
			return
		}
	}
	csr, err := cautils.NewCertificateRequestFromPEMFile(filepath.Join(cautils.DetermineCrtDir(os.Getenv), "client.csr"))
	if err != nil {
		log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
		model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
		return
	}
	pem, _ := csr.ToPEM()
	data := make(url.Values)
	data.Add("csr", string(pem))
	httpreq, err := http.NewRequest("POST", "http://"+credentials.Servers[0].Address+":7070/saltboot/csr/client", strings.NewReader(data.Encode()))
	httpreq.Header.Add("Authorization", "Token "+*authToken)
	httpreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, _ := http.DefaultClient.Do(httpreq)
	crtBytes, _ := ioutil.ReadAll(resp.Body)
	crt, err := cautils.NewCertificateFromPEM(crtBytes)
	if err != nil {
		log.Printf("[ClientCredsHandler] [ERROR]: %s", err.Error())
		model.Response{Status: err.Error()}.WriteInternalServerErrorHttp(w)
		return
	}
	err = crt.ToPEMFile(filepath.Join(cautils.DetermineCrtDir(os.Getenv), "client.crt"))
	model.Response{Status: "OK"}.WriteHttp(w)
	return
}

func ClientCredsDistributeHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("[ClientCredsDistributeHandler] execute distribute hostname request")

	decoder := json.NewDecoder(req.Body)
	var credentials Credentials
	err := decoder.Decode(&credentials)
	if err != nil {
		log.Printf("[ClientCredsDistributeHandler] [ERROR] couldn't decode json: %s", err)
		model.Response{Status: err.Error()}.WriteBadRequestHttp(w)
		return
	}

	user, pass := GetAuthUserPass(req)
	responses := credentials.DistributeClientCredentials(user, pass)
	cResp := model.Responses{Responses: responses}
	log.Printf("[ClientCredsDistributeHandler] distribute request executed: %s" + cResp.String())
	json.NewEncoder(w).Encode(cResp)
}

func (credentials *Credentials) DistributeClientCredentials(user string, pass string) []model.Response {
	log.Printf("[Clients.DistributeClientCredentials] Request: %v", credentials)
	credReqs := make([][]byte, 0)
	var pubIP *string
	for idx, _ := range credentials.Servers {
		if idx == 0 {
			pubIP = credentials.PublicIP
		} else {
			pubIP = nil
		}
		tmpToken := cautils.NewToken(10, 10)
		cautils.Store(filepath.Join(cautils.DetermineCrtDir(os.Getenv), "tokens", tmpToken.RandomHash), tmpToken)
		credReq := Credentials{
			Clients: Clients{
				Servers: credentials.Servers,
			},
			PublicIP:  pubIP,
			AuthToken: &tmpToken.RandomHash,
		}
		jsonBody, _ := json.Marshal(credReq)
		credReqs = append(credReqs, jsonBody)
	}

	resp := distributeImpl(Distribute, []string{credentials.Servers[0].Address}, credReqs[0], ClientCredsEP, user, pass)
	for _, r := range resp {
		if r.StatusCode != http.StatusOK {
			return resp
		}
	}

	return append(resp, distributeImplSlice(Distribute, credentials.Clients.Clients, credReqs, ClientCredsEP, user, pass)...)
}
