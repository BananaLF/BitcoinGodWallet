package main

import (
	"encoding/csv"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
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
	address string
	amount string
	txid string
	blockHash string
}

const windows  = 10

func main() {
	// create new client instance
	client, err := btcrpcclient.New(&btcrpcclient.ConnConfig{
		HTTPPostMode: true,
		DisableTLS:   true,
		Host:         "127.0.0.1:18886",
		User:         "123",
		Pass:         "123",
	}, nil)
	if err != nil {
		log.Fatalf("error creating new btc client: %v", err)
		return
	}

	cfg := chaincfg.TestNet3Params
	//cfg.PubKeyHashAddrID = 97
	//cfg.ScriptHashAddrID = 23

	list,err := ReadCSV()
	if err != nil {
		log.Fatalf("error ReadCSV: %v", err)
		return
	}
	length := len(list)
	for start := 0; start < length; start += windows {
		receivers := make(map[btcutil.Address]btcutil.Amount)
		end := start + windows
		if end > length {
			end = length
		}
		if err := ConvertToMap(list[start:end],receivers,&cfg);err != nil {
			log.Fatalf("error ConvertToMap: %v", err)
			return
		}

		if err := SendMany(client,receivers,&list,start,end);err != nil {
			log.Fatalf("error SendMany: %v", err)
			return
		}
	}

	errAddress := 0
	for i := range list {
		if len(list[i].txid) == 0 {
			errAddress += 1
		}
	}
	log.Println("success address: %v,error address:%v",(len(list)-errAddress),errAddress)
	if err := WriteCSVTx(list); err != nil {
		log.Fatalf("error WriteCSVTx: %v", err)
		return
	}

	time.Sleep(time.Duration(150)*time.Second)
	hasblock := make(map[string]string)
	for i := range list {
		for {
			if value,ok := hasblock[list[i].txid]; ok {
				list[i].blockHash = value
				continue
			}

			txSha,err := chainhash.NewHashFromStr(list[i].txid)
			if err != nil {
				continue
			}
			txResult,err := client.GetTransaction(txSha)
			if err != nil {
				log.Fatalf("error GetTransaction: %v", err)
				continue
			}
			if txResult.Confirmations != 0 {
				list[i].blockHash = txResult.BlockHash
				break
			}
			time.Sleep(time.Duration(150)*time.Second)
		}
	}

	if err := WriteCSV(list); err != nil {
		log.Fatalf("error WriteCSV: %v", err)
		return
	}
}

func ReadCSV() ([]Result,error) {
	dat, err := ioutil.ReadFile("/Users/lifei/Desktop/godAddress.csv")
	if err != nil {
		log.Fatalf("error read csv file: %v", err)
		return nil,err
	}
	r := csv.NewReader(strings.NewReader(string(dat[:])))
	list := make([]Result,0)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error read csv file line: %v", err)
			return nil,err
		}
		result := Result{address:record[1],amount:record[0]}
		list = append(list,result)
	}
	return list,nil
}

func WriteCSV(result []Result) error {
	f, err := os.Create("/Users/lifei/Desktop/testblock.csv")//创建文件
	if err != nil {
		panic(err)
		return err
	}
	defer f.Close()

	f.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM

	w := csv.NewWriter(f)//创建一个新的写入文件流
	data := make([][]string,0)
	for i := range result {
		line := make([]string,0)
		line = append(line,result[i].amount) //amount
		line = append(line,result[i].address) //address
		line = append(line,result[i].txid) //txid
		line = append(line,result[i].blockHash) //txid
		data = append(data,line)
	}
	w.WriteAll(data)//写入数据
	w.Flush()
	return nil
}

func WriteCSVTx(result []Result) error {
	f, err := os.Create("/Users/lifei/Desktop/testtx.csv")//创建文件
	if err != nil {
		panic(err)
		return err
	}
	defer f.Close()

	f.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM

	w := csv.NewWriter(f)//创建一个新的写入文件流
	data := make([][]string,0)
	for i := range result {
		line := make([]string,0)
		line = append(line,result[i].amount) //amount
		line = append(line,result[i].address) //address
		line = append(line,result[i].txid) //txid
		data = append(data,line)
	}
	w.WriteAll(data)//写入数据
	w.Flush()
	return nil
}

func ConvertToMap(list []Result,receivers map[btcutil.Address]btcutil.Amount,cfg *chaincfg.Params) error {
	temp := make(map[string]btcutil.Amount)
	for i := range list {
		money,err  := strconv.ParseFloat(list[i].amount,64)
		if err != nil {
			log.Fatalf("error read csv file convert amount int: %v", err)
		}
		amount, err := btcutil.NewAmount(money)
		if err != nil {
			log.Fatalf("error read csv file convert amount float64: %v", err)
		}
		if _,ok := temp[list[i].address]; ok {
			temp[list[i].address] += amount
		} else {
			temp[list[i].address] = amount
		}
	}

	for k,v := range temp {
		receiver, err := btcutil.DecodeAddress(k, cfg)
		if err != nil {
			continue
			log.Fatalf("error read csv file convert address: %v", err)
			return err
		}

		receivers[receiver] = v
	}

	return nil
}

func SendMany(client *btcrpcclient.Client,receivers map[btcutil.Address]btcutil.Amount,list *[]Result,start,end int) error {
	// create and send the sendMany tx
	txSha, err := client.SendMany("", receivers)
	if err != nil {
		log.Fatalf("error sendMany: %v", err)
		return err
	}
	//txSha := chainhash.HashH([]byte(string(start)))
	//for k,v := range receivers {
	//	log.Printf("address:%s,amount%s", k.String(),v.String())
	//}
	for i := start;i < end; i++ {
		for k,_ := range receivers {
			if strings.Compare((*list)[i].address,k.String()) == 0 {
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