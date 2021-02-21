package main

type DslStats struct {
	Down               string
	Up                 string
	LinkUptime         string
	Retrains           string
	LossOfPowerLink    string
	LossOfSignalLink   string
	LinkTrainErrors    string
	UnavailableSeconds string

	SNRDown string
	SNRUp   string

	AttenuationUp   string
	AttenuationDown string

	PowerUp   string
	PowerDown string

	PacketsDown string
	PacketsUp   string

	ErrorPacketsDown string
	ErrorPacketsUp   string

	CRCNearEnd string
	CRCFarEnd  string

	RSNearEnd string
	RSFarEnd  string
}