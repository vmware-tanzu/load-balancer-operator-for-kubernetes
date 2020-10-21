package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/avinetworks/sdk/go/clients"
	"github.com/avinetworks/sdk/go/session"
)

func main() {
	var transport *http.Transport
	client, err := clients.NewAviClient("10.184.69.25", "admin", session.SetPassword("Admin!23"), session.SetNoControllerStatusCheck, session.SetTransport(transport), session.SetInsecure)
	if err != nil {
		log.Fatalf("%s\n", err.Error())
	}
	tenant, err := client.Tenant.GetByName("test")
	if err != nil {
		log.Fatalf("%s\n", err.Error())
	}
	fmt.Printf("%s, %s, %s\n", *tenant.Name, *tenant.URL, *tenant.UUID)
}
