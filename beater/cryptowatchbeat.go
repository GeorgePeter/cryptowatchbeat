package beater

import (
	"flag"
	"strconv"
	"time"
	"github.com/GeorgePeter/cryptowatch-go"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/GeorgePeter/cryptowatchbeat/config"
	"github.com/GeorgePeter/cryptowatchbeat/persistency"
)

var mapFile = flag.String("p", "cryptowatchmap.json", "Path to the persistency map json file")

type Cryptowatchbeat struct {
	beatConfig     *config.CryptwatchConfig
	done           chan struct{}
	period         time.Duration
	api            *cryptowatch.CryptwatchClient
	collecting     bool
	client         publisher.Client
	cryptowatchMap *persistency.StringMap
}

// Creates beater
func New() *Cryptowatchbeat {
	return &Cryptowatchbeat{
		done: make(chan struct{}),
	}
}

/// *** Beater interface methods ***///

func (tb *Cryptowatchbeat) HandleFlags(b *beat.Beat) {
	logp.Info("Handling beat flags")

	tb.cryptowatchMap = persistency.NewStringMap()
	tb.cryptowatchMap.Load(*mapFile)
}

func (bt *Cryptowatchbeat) Config(b *beat.Beat) error {
	logp.Info("Loading configuration")

	// Load beater configuration
	var err error
	bt.beatConfig, err = config.NewCryptowatchbeatConfig()
	if err != nil {
		return err
	}

	return nil
}

func (bt *Cryptowatchbeat) Setup(b *beat.Beat) error {
	logp.Info("Setup waitduration")

	bt.client = b.Events

	var err error
	bt.period, err = time.ParseDuration(*bt.beatConfig.Period)
	if err != nil {
		return err
	}

	bt.api = new(cryptowatch.CryptwatchClient)

	return nil
}

func (bt *Cryptowatchbeat) Run(b *beat.Beat) error {
	logp.Info("cryptowatchbeat is running! Hit CTRL-C to stop it.")

	var err error
	ticker := time.NewTicker(bt.period)

	defer ticker.Stop()

	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
			if !bt.collecting {
				err = bt.collectRates()
				if err != nil {
					return err
				}
			}
		}
	}
}

func (bt *Cryptowatchbeat) Cleanup(b *beat.Beat) error {
	logp.Info("Cleanup api")

	//bt.api.Close()
	return nil
}

func (bt *Cryptowatchbeat) Stop() {
	logp.Info("Ctrl-C was hit, stopping.")

	close(bt.done)
}

func (bt *Cryptowatchbeat) collectRates() error {
	logp.Info("Collecting rates")

	bt.collecting = true
	defer func() {
		bt.collecting = false
	}()

	sync, err, processed := make(chan byte), make(chan error), 0

	var toProcess = 0
	for _, market := range *bt.beatConfig.Cryptowatch.Markets {
		toProcess += len(*market.Pairs)
	}

	for _, market := range *bt.beatConfig.Cryptowatch.Markets {
		go bt.processMarkets(market, sync, err)
	}

	for {
		select {
		case <-sync:
			processed++
			if processed == toProcess {
				return nil
			}
		case e := <-err:
			return e
		}
	}

	return nil
}
func (bt *Cryptowatchbeat) processMarkets(market config.Exchange, sync chan byte, err chan error) {
	for _, pair := range *market.Pairs {
		logp.Info("process market %v", market.Name)
		bt.processMarketAssets(market.Name, pair, sync, err)
	}
}

func (bt *Cryptowatchbeat) processResponse(result cryptowatch.OHLC, key string, market string, assetPair string) {
	var lastCloseTime = int64(0)

	for _, rate := range result {
		for _, summary := range rate {
			event := common.MapStr{
				"@timestamp": common.Time(summary.CloseTime),
				"type":       "ohlc",
				"volume":     summary.Volume,
				"high":       summary.HighPrice,
				"low":        summary.LowPrice,
				"open":       summary.OpenPrice,
				"close":      summary.ClosePrice,
				"asset":      assetPair,
				"exchange":   market,
			}
			if lastCloseTime == 0 || summary.CloseTimestamp > lastCloseTime {
				lastCloseTime = summary.CloseTimestamp
			}
			logp.Info("received Event - {high: %v volume: %v}", summary.HighPrice)
			bt.client.PublishEvent(event)
		}
	}
	// save the last id
	if len(result) >= 1 {
		bt.cryptowatchMap.Set(key, strconv.FormatInt(lastCloseTime, 10))
	}
}

func (bt *Cryptowatchbeat) processMarketAssets(market string, assetPair string, sync chan byte, err chan error) {
	// name in this case will be combination of asset-pair and market
	logp.Info("Collecting Rates for '%v' and pair %v", market, assetPair)

	var e error
	key := market + assetPair
	var after = bt.beatConfig.Cryptowatch.After
	if bt.cryptowatchMap.Contains(key) {
		after, e = strconv.ParseInt(bt.cryptowatchMap.Get(key), 10, 64)
	}
	if e != nil {
		err <- e
		return
	}
	periods := bt.beatConfig.Cryptowatch.Periods
	exchange := cryptowatch.Market{Exchange: market, CurrencyPair: assetPair}
	params := cryptowatch.OHLCParams{After: after, Periods: periods}
	result, e := bt.api.OHLC(exchange, params)

	if e != nil {
		logp.Critical("Evil error happend: %v", e)
		err <- e
		return
	}
	bt.processResponse(*result, key, market, assetPair)

	sync <- 1
}
