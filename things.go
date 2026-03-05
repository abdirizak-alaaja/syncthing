package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const nouns = `
Geel
Libaax
Dibi
Faras
Ari
Shabeel
Gorgor
Maroodi
Dawaco
Waraabe
Buur
Bad
Doox
Xiddig
Dayax
Qorrax
Roob
Dabayl
Daruur
Hillaac
Webi
Biyo
Har
Dhul
Carro
Ciid
Beer
Geed
Ubax
Caleen
Qalin
Buug
Kombuyuutar
Server
Disk
Cloud
Network
Packet
Stream
Node
Cluster
System
Engine
Machine
Bridge
Tunnel
Port
Signal
Wave
Matrix
Vector
Orbit
Galaxy
Comet
Asteroid
Planet
`

const adjectives = `
Degdeg
Adag
Xoog
Degan
Fudud
Qurux
Weyn
Yar
Fiican
Hufan
Nadiif
Cajiib
Adkaysi
Furan
Deggan
Dheer
Gaaban
Cusub
Qadiim
Dhakhso
Awood
Fahmo
Xariif
Fudud
LaYaab
Hagaagsan
Sugan
Deggan
QuruxBadan
NadiifBadan
AdagBad
XoogBad
DegdegBadan
Xasiloon
Firfircoon
Xamaasad
Dabacsan
Fayow
Caadi
Cilmi
Caan
Cad
Madow
Buluug
Cas
Cagaar
Huruud
Sare
Hoose
Fog
Dhow
Ballaaran
Cidhiidhi
`

var (
	nounList      = parseList(nouns)
	adjectiveList = parseList(adjectives)
)

func parseList(s string) []string {
	lines := strings.Split(s, "\n")
	var result []string

	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			result = append(result, l)
		}
	}

	return result
}

func randIndex(max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}

func GenerateThing() string {
	number := randIndex(90) + 10

	noun := nounList[randIndex(len(nounList))]
	adj1 := adjectiveList[randIndex(len(adjectiveList))]

	adj2 := adj1
	for adj2 == adj1 {
		adj2 = adjectiveList[randIndex(len(adjectiveList))]
	}

	return fmt.Sprintf("%s%s%s%d", adj1, adj2, noun, number)
}
