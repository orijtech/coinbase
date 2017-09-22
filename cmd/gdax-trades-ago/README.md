# gdax-trades-ago

## Installing it
```shell
go get -u github.com/orijtech/coinbbase/cmd/gdax-trades-ago
```

## Using it
You can optionally set:
* dur-ago: The duration to go back behind. Duration values are defined at https://golang.org/pkg/time/#ParseDuration
So valid dur-ago suffixes are "ns", "us" (or "µs"), "ms", "s", "m", "h".
For example: -dur-ago 86h20m
* product: The <from>-<to> currency pair defined at https://docs.gdax.com/#get-products
For example: -product BTC-USD

```shell
gdax-trades-ago -dur-ago 2h -product BTC-USD && head -n 10 data.csv 
2017/09/21 21:47:42 Flushed page: #1
data,timeEpoch,high,low,open,close,volume
2017-09-21T18:47:00.00000Z,1506041220,3643.6000,3643.6000,3643.6000,3643.6000,0.9226
2017-09-21T18:46:00.00000Z,1506041160,3643.5900,3643.9900,3643.5900,3643.6000,5.9599
2017-09-21T18:45:00.00000Z,1506041100,3643.5900,3644.0000,3644.0000,3643.9600,2.5004
2017-09-21T18:44:00.00000Z,1506041040,3643.9900,3644.0000,3644.0000,3644.0000,2.1227
2017-09-21T18:43:00.00000Z,1506040980,3644.0000,3644.1600,3644.1600,3644.0000,5.6206
2017-09-21T18:42:00.00000Z,1506040920,3642.1200,3644.5800,3644.5800,3644.1600,6.1915
2017-09-21T18:41:00.00000Z,1506040860,3640.8400,3646.6100,3646.6100,3644.5800,8.6110
2017-09-21T18:40:00.00000Z,1506040800,3646.6000,3647.0500,3647.0500,3646.6100,5.5431
2017-09-21T18:39:00.00000Z,1506040740,3647.0000,3649.8500,3649.7000,3647.0500,6.2359
```
