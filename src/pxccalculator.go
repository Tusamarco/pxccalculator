package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"pxccalculator/src/Global"
	"pxccalculator/src/Objects"
	"strconv"
)

func main() {

	var (
		port    int
		ip      string
		helpB   bool
		version bool
		help    Global.HelpText
	)
	port = *flag.Int("port", 8080, "Port to serve")
	ip = *flag.String("address", "0.0.0.0", "Ip address")
	flag.BoolVar(&helpB, "help", false, "for help")
	flag.BoolVar(&version, "version", false, "to get product version")
	flag.Parse()

	var versionS = "0.0.1"
	//initialize help

	//just check if we need to pass version or help
	if version {
		fmt.Println("PXC calculator for Operator Version: ", versionS)
		exitWithCode(0)
	} else if helpB {
		flag.PrintDefaults()
		fmt.Fprintf(os.Stdout, "%s\n", help.GetHelpText())
		exitWithCode(0)
	}

	//set log level
	log.SetLevel(log.DebugLevel)

	//set server address (need to come from configuration parameter)
	server := http.Server{Addr: ip + ":" + strconv.Itoa(port)}

	//define API handlers
	http.HandleFunc("/calculator", handleRequestCalculator)
	http.HandleFunc("/supported", handleRequestSupported)
	server.ListenAndServe()
}

// When need to calculate we (for the moment) just catch the request and pass over
func handleRequestCalculator(writer http.ResponseWriter, request *http.Request) {
	var err error
	switch request.Method {
	case "GET":
		err = handleGetCalculate(writer, request)

	}
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		log.Error(err)
	}
}

func handleRequestSupported(writer http.ResponseWriter, request *http.Request) {
	var err error
	switch request.Method {
	case "GET":
		err = handleGetSupported(writer, request)
	}
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		log.Error(err)
	}
}

// here we return a configuration answering to a request like:
// { "dimension":  {"id": 3, "name": "XSmall",  "cpu": 1000}, "loadtype":  {"id": 2, "name": "Mainly Reads"}, "connections": 50}

func handleGetCalculate(writer http.ResponseWriter, request *http.Request) error {
	// message object to pass back
	var responseMsg Objects.ResponseMessage
	var family Objects.Family
	var conf Objects.Configuration
	var families map[string]Objects.Family
	var ConfRequest Objects.ConfigurationRequest

	//if we do not have a real request we return a message with the info
	len := request.ContentLength
	if len <= 0 {
		err := returnErrorMessage(writer, request, &ConfRequest, responseMsg, families, "Empty request")
		if err != nil {
			return err
		}
		return nil
	}
	body := make([]byte, len)
	request.Body.Read(body)

	// we need to process the request and get the values
	json.Unmarshal(body, &ConfRequest)

	// Before going to the configurator we check the incoming request and IF is not ok we return an error message
	if ConfRequest.Dimension.Id == 0 || ConfRequest.LoadType.Id == 0 {
		err := returnErrorMessage(writer, request, &ConfRequest, responseMsg, families, "Possible Malformed request "+string(body[:]))
		if err != nil {
			return err
		}
		return nil
	}

	// create and init all the different params organized by families
	families = family.Init()
	conf.Init()

	// initialize the configurator (where all the things happens)
	var c Configurator
	c.init(ConfRequest, families, conf)

	// here is the calculation step
	c.ProcessRequest()

	err := ReturnResponse(writer, request, &ConfRequest, responseMsg, families)
	if err != nil {
		return err
	}

	return nil

	//fmt.Fprintf(os.Stdout, "\n%s\n", body)

}

func ReturnResponse(writer http.ResponseWriter, request *http.Request, ConfRequest *Objects.ConfigurationRequest, message Objects.ResponseMessage, families map[string]Objects.Family) error {
	//output, err := json.Marshal(&conf)

	messageStream, err := json.MarshalIndent(message, "", "  ")
	// we store incoming request for reference when passing back the configuration
	processedRequest, err := json.MarshalIndent(&ConfRequest, "", "  ")

	// We transform to Json all the calculated params
	output, err := json.MarshalIndent(&families, "", "  ")

	// Concatenate all into a single output
	var b bytes.Buffer
	b.WriteString(`{"request": {`)
	b.WriteString(`,"message":`)
	b.Write(messageStream)
	b.WriteString(`,"incoming":`)
	b.Write(processedRequest)
	b.WriteString(`,"answer":`)
	b.Write(output)
	b.WriteString("}}")

	if err != nil {
		return err
	}

	// Return the information
	writer.Header().Set("Content/Type", "application/json")
	writer.Write(b.Bytes())
	return nil
}

func handleGetSupported(writer http.ResponseWriter, request *http.Request) error {
	var conf Objects.Configuration
	conf.Init()
	//output, err := json.Marshal(&conf)
	output, err := json.MarshalIndent(&conf, "", "  ")

	if err != nil {
		return err
	}

	writer.Header().Set("Content/Type", "application/json")
	writer.Write(output)
	return nil
}

func exitWithCode(errorCode int) {
	log.Debug("Exiting execution with code ", errorCode)
	os.Exit(errorCode)
}

func returnErrorMessage(writer http.ResponseWriter, request *http.Request, ConfRequest *Objects.ConfigurationRequest, message Objects.ResponseMessage, families map[string]Objects.Family, errorMessage string) error {
	message.MType = Objects.ERROREXEC_I
	message.MName = "Invalid incoming request"
	message.MText = fmt.Sprintf(message.GetMessageText(message.MType), errorMessage)
	err := ReturnResponse(writer, request, ConfRequest, message, families)
	if err != nil {
		return err
	}

	return nil
}
