package cautils

import (
	"encoding/pem"
	"io/ioutil"
	"os"
	"log"
)

const (
  caLocKey         = "SALTBOOT_CA"
	defaultCaLoc     = "./ca"
)

type ToPem interface {
	ToPEM() ([]byte, error)
}

func newFromPEMFile(filename string, creator func([]byte) (interface{}, error)) (interface{}, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return creator(data)
}

func toPemImpl(pemType string, derBytes []byte) ([]byte, error) {
	pemBlock := &pem.Block{
		Type:  pemType,
		Bytes: derBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	return pemBytes, nil
}

func toPemFileImpl(source ToPem, filename string) error {
	pemBytes, err := source.ToPEM()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, pemBytes, 0400)
}


func DetermineCaRootDir(getEnv func(key string) string) string {
	caLocation := os.Getenv(caLocKey)
	log.Printf("[determineCaRootDir] CA_ROOT_DIR: %s", caLocation)
	if len(caLocation) == 0 {
		caLocation = defaultCaLoc
	}
	return caLocation
}