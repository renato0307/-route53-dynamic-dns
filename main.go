package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	externalip "github.com/glendc/go-external-ip"
)

var name string
var target string
var ttl int64
var weight = int64(1)
var zoneID string
var refreshInterval time.Duration
var setIdentifier string

func init() {
	rand.Seed(time.Now().UnixNano())

	flag.StringVar(&name, "d", "", "domain name")
	flag.StringVar(&zoneID, "z", "", "AWS Zone Id for domain")
	flag.Int64Var(&ttl, "ttl", int64(60), "TTL for DNS Cache")
	flag.DurationVar(&refreshInterval, "r", time.Duration(1), "refresh interval in minutes")
}

func main() {
	flag.Parse()
	if name == "" || zoneID == "" {
		log.Println(fmt.Errorf("incomplete arguments: d: %s, t: %s, z: %s", name, target, zoneID))
		flag.PrintDefaults()
		return
	}
	sess, err := session.NewSession()
	if err != nil {
		log.Println("failed to create session,", err)
		return
	}

	setIdentifier = randStringRunes(10)
	svc := route53.New(sess)
	for {
		consensus := externalip.DefaultConsensus(nil, nil)
		ip, err := consensus.ExternalIP()
		if err != nil {
			log.Fatal()
		}

		target = ip.String()
		log.Printf("ip address is %s\n", target)
		createARecord(svc)

		log.Printf("waiting for %d minutes", refreshInterval)
		time.Sleep(refreshInterval * time.Minute)
	}
}

func createARecord(svc *route53.Route53) {
	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(name),
						Type: aws.String("A"),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(target),
							},
						},
						TTL:           aws.Int64(ttl),
						Weight:        aws.Int64(weight),
						SetIdentifier: aws.String(setIdentifier),
					},
				},
			},
			Comment: aws.String("Dynamic DNS update."),
		},
		HostedZoneId: aws.String(zoneID),
	}
	resp, err := svc.ChangeResourceRecordSets(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		log.Fatal(err.Error())
		return
	}

	// Pretty-print the response data.
	log.Println("change response:")
	log.Println(resp)
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
