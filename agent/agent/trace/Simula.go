package trace

import (
	"time"

	"github.com/whatap/golib/lang/pack/udp"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/keygen"
)

func StartSimula() {
	go func() {

		for {
			time.Sleep(3000 * time.Millisecond)

			tx_s := udp.NewUdpTxStartPack()
			tx_s.Time = dateutil.Now()
			tx_s.Txid = keygen.Next()
			tx_s.Uri = "simula.php"
			tx_s.Process()
			TxPut(tx_s)
			//			time.Sleep(1*time.Millisecond)

			tx_q := udp.NewUdpTxSqlPack()
			tx_q.Time = dateutil.Now()
			tx_q.Elapsed = int32(10)
			tx_q.Txid = tx_s.Txid
			tx_q.Sql = "select * from php"
			tx_s.Process()
			TxPut(tx_q)
			time.Sleep(20 * time.Millisecond)

			tx_e := udp.NewUdpTxEndPack()
			tx_e.Time = dateutil.Now()
			tx_e.Txid = tx_s.Txid
			tx_e.Uri = ""
			tx_e.Process()
			TxPut(tx_e)

			//	time.Sleep(200*time.Millisecond)
		}

	}()
}
