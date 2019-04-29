package main

import (
	"encoding/csv"
	"flag"
	"github.com/btcsuite/btcd/chaincfg"
	btcrpcclient "github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type Result struct {
	address   string
	amount    string
	txid      string
	blockHash string
}

var windows = 1000
var path = "/Users/lifei"

func main() {
	rpcuser := flag.String("rpcuser", "123", "rpcuser")
	rpcpassword := flag.String("rpcpassword", "123", "rpcpassword")
	rpcaddr := flag.String("rpcaddr", "127.0.0.1:18886", "rpcaddr")
	rpcwindows := flag.Int("windows", 1000, "windows")
	isMainNet := flag.Bool("ismainnet", true, "ismainnet")
	datadir := flag.String("datadir", "/Users/lifei/Desktop", "datadir")

	flag.Parse()
	path = *datadir
	windows = *rpcwindows
	// create new client instance
	client, err := btcrpcclient.New(&btcrpcclient.ConnConfig{
		HTTPPostMode: true,
		DisableTLS:   true,
		Host:         *rpcaddr,
		User:         *rpcuser,
		Pass:         *rpcpassword,
	}, nil)
	if err != nil {
		log.Fatalf("error creating new btc client: %v", err)
		return
	}

	var cfg chaincfg.Params
	if *isMainNet {
		cfg = chaincfg.MainNetParams
		cfg.PubKeyHashAddrID = 97
		cfg.ScriptHashAddrID = 23
	} else {
		cfg = chaincfg.TestNet3Params
	}

	list, err := ReadCSV()
	if err != nil {
		log.Fatalf("error ReadCSV: %v", err)
		return
	}
	length := len(list)
	index := 0
	for start := 0; start < length; start += windows {
		receivers := make(map[btcutil.Address]btcutil.Amount)
		end := start + windows
		if end > length {
			end = length
		}
		temp, err := ConvertToMap(list[start:end], receivers, &cfg)
		if err != nil {
			log.Fatalf("error ConvertToMap: %v", err)
			return
		}
		if index < 5 {
			addresses := DecodeTransaction(client, index)
			if len(addresses) != len(temp) {
				log.Println("error Transaction map len: %d list lent:%d", len(temp), len(addresses))
				log.Println("error Transaction map len: %d", index)
				return
			}
			for i := range addresses {
				value, ok := temp[addresses[i].address]
				if ok {
					if strings.Compare(value.String(), addresses[i].amount) != 0 {
						log.Fatalf("error Transaction: %d", index)
						return
					}
				} else {
					log.Fatalf("error Transaction: %d", index, "address", addresses[i].address, "amount", addresses[i].amount)
					return
				}

			}
			for k := start; k < end; k++ {
				if _, err := btcutil.DecodeAddress(list[k].address, &cfg); err != nil {
					continue
				}
				list[k].txid = addresses[0].txid
				if strings.Compare("47ae26da366d7a4de4d778b890d306f30900609edd42a3f0859e0983c56fc45d", addresses[0].txid) == 0 ||
					strings.Compare("6cc9e0757e29b290681ee0c4b6741234ec1a9bc5d8a391b664e7e4177a33811b", addresses[0].txid) == 0 ||
					strings.Compare("7cc23a363d2198d470a8eddd43e46c5d6ce9a2537bee4ea502d58ec29ce445a5", addresses[0].txid) == 0 {
					list[k].blockHash = "c37eab402bd3b017b5a940863b2d9bc246e23193a60ebf0af118354338aa4c6d"
				} else if strings.Compare("bb789bd246a82b0592b3936b98e88cb4dfb960e78b0b9d0799c1fb28650fdfd9", addresses[0].txid) == 0 {
					list[k].blockHash = "172ee331fcc1a28a26a76d5243221952927bff81761829a8aa13a0c4253b2151"
				} else if strings.Compare("e10cfcc3b2567f2cef07847cb68d353b91815a1441e8237ee4cf570aeb980221", addresses[0].txid) == 0 {
					list[k].blockHash = "4c40958a22af308c283c99512cfe3043cc96738d3cf217a4ff26c69987aeb12c"
				}
			}
			index++
		} else {
			if err := WriteLastCSV(list[start:end]); err != nil {
				return
			}
		}
	}

	if err := WriteCSV(list); err != nil {
		log.Fatalf("error WriteCSV: %v", err)
		return
	}

	/*errAddress := 0
	for i := range list {
		if len(list[i].txid) == 0 {
			errAddress += 1
		}
	}
	log.Println("success address: %v,error address:%v", (len(list) - errAddress), errAddress)
	if err := WriteCSVTx(list); err != nil {
		log.Fatalf("error WriteCSVTx: %v", err)
		return
	}

	if err := WriteCSV(list); err != nil {
		log.Fatalf("error WriteCSV: %v", err)
		return
	}*/
}

func DecodeTransaction(client *btcrpcclient.Client, index int) []Result {
	t, err := client.DecodeRawTransaction(transaction[index])
	if err != nil {
		log.Fatalf("error DecodeRawTransaction: %v", err)
		return nil
	}
	results := make([]Result, 0)
	for i := range t.Vout {
		if t.Vout[i].Value < 100 {
			amount, _ := btcutil.NewAmount(t.Vout[i].Value)
			result := Result{address: t.Vout[i].ScriptPubKey.Addresses[0], amount: amount.String(), txid: t.Hash, blockHash: t.BlockHash}
			results = append(results, result)
		}
	}
	return results
}

func ReadCSV() ([]Result, error) {
	dat, err := ioutil.ReadFile(path + "/godAddress.csv")
	if err != nil {
		log.Fatalf("error read csv file: %v", err)
		return nil, err
	}
	r := csv.NewReader(strings.NewReader(string(dat[:])))
	list := make([]Result, 0)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error read csv file line: %v", err)
			return nil, err
		}
		result := Result{address: record[1], amount: record[0]}
		list = append(list, result)
	}
	return list, nil
}

func WriteCSV(result []Result) error {
	f, err := os.Create(path + "/testblock.csv") //创建文件
	if err != nil {
		panic(err)
		return err
	}
	defer f.Close()

	f.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM

	w := csv.NewWriter(f) //创建一个新的写入文件流
	data := make([][]string, 0)
	for i := range result {
		line := make([]string, 0)
		line = append(line, result[i].amount)    //amount
		line = append(line, result[i].address)   //address
		line = append(line, result[i].txid)      //txid
		line = append(line, result[i].blockHash) //txid
		data = append(data, line)
	}
	w.WriteAll(data) //写入数据
	w.Flush()
	return nil
}

func WriteLastCSV(result []Result) error {
	f, err := os.Create(path + "/testlast.csv") //创建文件
	if err != nil {
		panic(err)
		return err
	}
	defer f.Close()

	f.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM

	w := csv.NewWriter(f) //创建一个新的写入文件流
	data := make([][]string, 0)
	for i := range result {
		line := make([]string, 0)
		line = append(line, result[i].amount)  //amount
		line = append(line, result[i].address) //address
		data = append(data, line)
	}
	w.WriteAll(data) //写入数据
	w.Flush()
	return nil
}

func WriteCSVTx(result []Result) error {
	f, err := os.Create(path + "/testtx.csv") //创建文件
	if err != nil {
		panic(err)
		return err
	}
	defer f.Close()

	f.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM

	w := csv.NewWriter(f) //创建一个新的写入文件流
	data := make([][]string, 0)
	for i := range result {
		line := make([]string, 0)
		line = append(line, result[i].amount)  //amount
		line = append(line, result[i].address) //address
		line = append(line, result[i].txid)    //txid
		data = append(data, line)
	}
	w.WriteAll(data) //写入数据
	w.Flush()
	return nil
}

func ConvertToMap(list []Result, receivers map[btcutil.Address]btcutil.Amount, cfg *chaincfg.Params) (map[string]btcutil.Amount, error) {
	temp := make(map[string]btcutil.Amount)
	for i := range list {
		money, err := strconv.ParseFloat(list[i].amount, 64)
		if err != nil {
			log.Fatalf("error read csv file convert amount int: %v", err)
			return nil, err
		}
		amount, err := btcutil.NewAmount(money)
		if err != nil {
			log.Fatalf("error read csv file convert amount float64: %v", err)
			return nil, err
		}
		if _, err := btcutil.DecodeAddress(list[i].address, cfg); err != nil {
			continue
		}
		if _, ok := temp[list[i].address]; ok {
			temp[list[i].address] += amount
		} else {
			temp[list[i].address] = amount
		}
	}

	for k, v := range temp {
		receiver, err := btcutil.DecodeAddress(k, cfg)
		if err != nil {
			continue
			log.Fatalf("error read csv file convert address: %v", err)
			return temp, err
		}

		receivers[receiver] = v
	}

	return temp, nil
}

func SendMany(client *btcrpcclient.Client, receivers map[btcutil.Address]btcutil.Amount, list *[]Result, start, end int) error {
	// create and send the sendMany tx
	txSha, err := client.SendMany("", receivers)
	if err != nil {
		log.Fatalf("error sendMany: %v", err)
		return err
	}
	/*txSha := chainhash.HashH([]byte(string(start)))
	for k,v := range receivers {
		log.Printf("address:%s,amount%s", k.String(),v.String())
	}*/
	for i := start; i < end; i++ {
		for k, _ := range receivers {
			if strings.Compare((*list)[i].address, k.String()) == 0 {
				(*list)[i].txid = txSha.String()
				break
			}
		}
	}
	log.Printf("sendMany completed! tx sha is: %s", txSha.String())

	return nil
}

/*receiver2, err := btcutil.DecodeAddress("gD9X8YdJdocfjdXEYxfhwHyPpr7ptPVxzU", &cfg)
if err != nil {
	log.Fatalf("address receiver2 seems to be invalid: %v", err)
	return
}
log.Println("DecodeAddress",receiver2.String())
wif,err := btcutil.DecodeWIF("L3sKJRB2tGi3nsidn2eszZtTVYM6xHFSz6Kpx7DnCEs5NDn9Av6W")
addrpubkey,err := btcutil.NewAddressPubKey(wif.SerializePubKey(),&cfg)
if err != nil {
	log.Fatalf("error generate address: %v", err)
	return
}
log.Println("priv->address",addrpubkey.EncodeAddress())
if strings.Compare(addrpubkey.EncodeAddress(),receiver2.String()) == 0 {
	log.Println("Success")
}else  {
	log.Println("Faild")
}*/

/*// list accounts
accounts, err := client.ListAccounts()
if err != nil {
	log.Fatalf("error listing accounts: %v", err)
}
// iterate over accounts (map[string]btcutil.Amount) and write to stdout
for label, amount := range accounts {
	log.Printf("%s: %s", label, amount)
}

// prepare a sendMany transaction
receiver1, err := btcutil.DecodeAddress("1someAddressThatIsActuallyReal", &chaincfg.MainNetParams)
if err != nil {
	log.Fatalf("address receiver1 seems to be invalid: %v", err)
}
receiver2, err := btcutil.DecodeAddress("1anotherAddressThatsPrettyReal", &chaincfg.MainNetParams)
if err != nil {
	log.Fatalf("address receiver2 seems to be invalid: %v", err)
}
*/
