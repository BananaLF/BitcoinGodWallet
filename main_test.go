package main

import (
	"encoding/csv"
	"encoding/hex"
	"github.com/btcsuite/btcd/chaincfg"
	btcrpcclient "github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
)

var transaction = [][]byte{}

func TestResult(t *testing.T) {
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
	cfg := chaincfg.MainNetParams
	cfg.PubKeyHashAddrID = 97
	cfg.ScriptHashAddrID = 23

	//Read raw transaction
	for i := 1; i <= 10; i++ {
		rawTxfile := "/Users/lifei/Desktop/0502record" + "/tx" + strconv.Itoa(i)
		buffer, err := ReadRawTransaction(rawTxfile)
		if err != nil {
			t.Error("ReadRawTransaction file", i)
		}
		tx, err := hex.DecodeString(string(buffer))
		transaction = append(transaction, tx)
	}
	list, err := ReadResultCSV("/Users/lifei/Desktop/0502record/result.csv")
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
		temp, err := ConvertToMapForTest(list[start:end], receivers, &cfg)
		if err != nil {
			log.Fatalf("error ConvertToMap: %v", err)
			return
		}
		if index < len(transaction) {
			addresses, txid := DecodeTransaction(client, index)
			if (len(addresses) - 1) != len(temp) {
				t.Error("error Transaction map len: ", len(temp), " list lent:", len(addresses)-1)
				t.Error("error Transaction map len:", index)
				return
			}
			errAddress := 0
			for i := range addresses {
				value, ok := temp[addresses[i].address]
				if ok {
					if strings.Compare(value.String(), addresses[i].amount) != 0 {
						log.Fatalf("error Transaction: %d", index)
						return
					}
				} else {
					if errAddress >= 1 {
						t.Log("error Transaction:", index, "address", addresses[i].address, "amount", addresses[i].amount)
						return
					} else {
						errAddress++
					}
				}

			}
			for k := start; k < end; k++ {
				if _, err := btcutil.DecodeAddress(list[k].address, &cfg); err != nil {
					continue
				}
				if strings.Compare(txid, list[k].txid) != 0 {
					log.Fatalf("error txid is different: %d", index)
					return
				}
			}
			index++
		}
	}

	var sumOut float64 = 0
	var lastOut float64 = 0
	var sumValidAddress int = 0
	var sumInvaliAddress int = 0
	for i := 1; i < length; i++ {
		money, err := strconv.ParseFloat(list[i].amount, 64)
		if err != nil {
			log.Fatalf("error statictis: %v", err)
			return
		}

		if len(list[i].txid) == 0 {
			lastOut += money
			sumInvaliAddress++
		} else {
			sumOut += money
			sumValidAddress++
		}
	}
	t.Log("有效地址数:", sumValidAddress, "有效输出:", sumOut)
	t.Log("无效地址数:", sumInvaliAddress, "无效输出", lastOut)
}

func ReadRawTransaction(file string) ([]byte, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(f)
}

func DecodeTransaction(client *btcrpcclient.Client, index int) ([]Result, string) {
	t, err := client.DecodeRawTransaction(transaction[index])
	if err != nil {
		log.Fatalf("error DecodeRawTransaction:%s,%v", err, index)
		return nil, ""
	}
	results := make([]Result, 0)
	for i := range t.Vout {
		amount, _ := btcutil.NewAmount(t.Vout[i].Value)
		result := Result{address: t.Vout[i].ScriptPubKey.Addresses[0], amount: amount.String(), txid: t.Hash, blockHash: t.BlockHash}
		results = append(results, result)

	}
	return results, t.Hash
}

func ConvertToMapForTest(list []Result, receivers map[btcutil.Address]btcutil.Amount, cfg *chaincfg.Params) (map[string]btcutil.Amount, error) {
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
			return nil, err
		}
		_, err = btcutil.DecodeAddress(list[i].address, cfg)
		if err != nil {
			log.Println("error read csv file convert address:", list[i].address, "--error:", err)
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
			log.Println("error read csv file convert address:", k, "--error:", err)
			continue
		}

		receivers[receiver] = v
	}

	return temp, nil
}

func ReadResultCSV(file string) ([]Result, error) {
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
		result := Result{address: record[1], amount: record[0], txid: record[2]}
		list = append(list, result)
	}
	return list, nil
}
