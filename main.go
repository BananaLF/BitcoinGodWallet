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
	"time"
)

type Result struct {
	address   string
	amount    string
	txid      string
	blockHash string
}

var windows = 1000
var path = "/Users/lifei/Desktop"
var fileName = "/godAddress.csv"

func main() {

	rpcuser := flag.String("rpcuser", "123", "rpcuser")
	rpcpassword := flag.String("rpcpassword", "123", "rpcpassword")
	rpcaddr := flag.String("rpcaddr", "127.0.0.1:18886", "rpcaddr")
	rpcwindows := flag.Int("windows", 1000, "windows")
	isMainNet := flag.Bool("ismainnet", true, "ismainnet")
	datadir := flag.String("datadir", "/Users/lifei/Desktop", "datadir")
	datafile := flag.String("file", "/godAddress.csv", "godAddress.csv")
	flag.Parse()

	path = *datadir
	windows = *rpcwindows
	fileName = *datafile
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
	readFileName := path + fileName
	log.Println("Read data file")
	list, err := ReadCSV(readFileName)
	if err != nil {
		log.Fatalf("error ReadCSV: %v", err)
		return
	}

	log.Println("Start to send GOD")
	length := len(list)
	for start := 0; start < length; start += windows {
		receivers := make(map[btcutil.Address]btcutil.Amount)
		end := start + windows
		if end > length {
			end = length
		}
		if err := ConvertToMap(list[start:end], receivers, &cfg); err != nil {
			remainFileName := path + "/remainAddress" + strconv.Itoa(start+1) + "-" + strconv.Itoa(length) + ".csv"
			if err := WriteCSV(list[start:], remainFileName); err != nil {
				log.Fatalf("error remainFileName WriteCSVTx: %v", err)
				return
			}
			log.Fatalf("error ConvertToMap: %v", err)
			return
		}
		log.Println("send address from:", strconv.Itoa(start+1), "  to:", strconv.Itoa(end))
		if err := SendMany(client, receivers, &list, start, end); err != nil {
			remainFileName := path + "/remainAddress" + strconv.Itoa(start+1) + "-" + strconv.Itoa(length) + ".csv"
			if err := WriteCSV(list[start:], remainFileName); err != nil {
				log.Fatalf("error remainFileName WriteCSVTx: %v", err)
				return
			}
			log.Fatalf("error SendMany: %v", err)
			return
		}
		log.Println("waiting to write hasSend addresses file")
		hasSendFileName := path + "/hasSend" + strconv.Itoa(start+1) + "-" + strconv.Itoa(end) + ".csv"
		if err := WriteCSV(list[start:end], hasSendFileName); err != nil {
			log.Fatalf("error temptxid WriteCSVTx: %v", err)
			return
		}
		log.Println("send success")
	}

	log.Println("Success send GOD")
	log.Println("Writing to result file")
	resultFileName := path + "/result.csv"
	if err := WriteCSV(list, resultFileName); err != nil {
		log.Fatalf("error result WriteCSV: %v", err)
		return
	}

	log.Println("Statistics result")
	errAddress := 0
	for i := range list {
		if len(list[i].txid) == 0 {
			errAddress += 1
		}
	}
	log.Println("success send to addresses: %v,error address:%v", (len(list) - errAddress), errAddress)
}

func ConvertToMap(list []Result, receivers map[btcutil.Address]btcutil.Amount, cfg *chaincfg.Params) error {
	temp := make(map[string]btcutil.Amount)
	for i := range list {
		money, err := strconv.ParseFloat(list[i].amount, 64)
		if err != nil {
			log.Println("error read csv file convert amount int address:", list[i].String(), "--error:", err)
			continue
		}
		amount, err := btcutil.NewAmount(money)
		if err != nil {
			log.Println("error read csv file convert amount float64 address:", list[i].String(), "--error:", err)
			return err
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
			log.Println("error read csv file convert address:", k, "--error:", err)
			continue
		}

		receivers[receiver] = v
	}

	return nil
}

func SendMany(client *btcrpcclient.Client, receivers map[btcutil.Address]btcutil.Amount, list *[]Result, start, end int) error {
	// create and send the sendMany tx
	txSha, err := client.SendMany("", receivers)
	if err != nil {
		log.Println("error sendMany:", err)
		return err
	}
	//txSha := chainhash.HashH([]byte(string(start)))
	for i := start; i < end; i++ {
		for k, _ := range receivers {
			if strings.Compare((*list)[i].address, k.String()) == 0 {
				(*list)[i].txid = txSha.String()
				break
			}
		}
	}
	log.Println("tx:", txSha.String())
	log.Println("waiting for pushing to block")
	blockHash := "null"
	//wait transaction is pushed into block
	for {
		txResult, err := client.GetTransaction(txSha)
		if err != nil {
			return err
		}
		if txResult.Confirmations != 0 {
			blockHash = txResult.BlockHash
			break
		}
		time.Sleep(time.Duration(10) * time.Second)
	}
	log.Println("block:", blockHash)
	for i := start; i < end; i++ {
		for k, _ := range receivers {
			if strings.Compare((*list)[i].address, k.String()) == 0 {
				(*list)[i].blockHash = blockHash
				break
			}
		}
	}
	return nil
}

func ReadCSV(file string) ([]Result, error) {
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		log.Println("error read csv file: ", err)
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
			log.Println("error read csv file line:", err)
			return nil, err
		}
		result := Result{address: record[1], amount: record[0]}
		list = append(list, result)
	}
	return list, nil
}

func WriteCSV(result []Result, file string) error {
	f, err := os.Create(file) //创建文件
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

func (r *Result) String() string {
	return "{" + "address:" + r.address + ",amount:" + r.amount + ",txid:" + r.txid + ",blockhash" + r.blockHash + "}"
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
