package sqlutil

import (
	"bytes"
	"strings"

	_ "fmt"
	"github.com/whatap/golib/lang"
	_ "github.com/whatap/golib/util/dateutil"
	"github.com/whatap/go-api/agent/agent/config"
)

const (
	// 1
	NORMAL uint8 = 1
	// 2
	COMMENT uint8 = 2
	// 3
	ALPABET uint8 = 3
	// 4
	NUMBER uint8 = 4
	// 5
	QUTATION uint8 = 5
	// 6
	COLON uint8 = 6
	// 7
	DQUTATION uint8 = 7
)

type EscapeLiteralSQL struct {
	substitute          string
	substitute_num      string
	substitute_str_mode bool
	sql                 string
	pos                 int
	length              int
	parsedSql           *bytes.Buffer
	param               *bytes.Buffer
	status              uint8
	SqlType             uint8
	inclueComment       bool
}

func NewEscapeLiteralSQL(sql string) *EscapeLiteralSQL {
	p := new(EscapeLiteralSQL)

	p.substitute = "#"
	p.substitute_num = "#"
	p.substitute_str_mode = false
	p.inclueComment = true

	p.sql = sql
	p.length = len(p.sql)

	p.parsedSql = bytes.NewBuffer(make([]byte, len(p.sql)))
	p.parsedSql.Reset()
	p.param = bytes.NewBuffer(make([]byte, len(p.sql)))
	p.param.Reset()

	return p
}

func (this *EscapeLiteralSQL) Setsubstitute(chr string) *EscapeLiteralSQL {

	this.substitute = chr
	if this.substitute_str_mode {
		this.substitute_num = "'" + chr + "'"
	} else {
		this.substitute_num = this.substitute
	}
	return this

}

func (this *EscapeLiteralSQL) SetSubstituteStringMode(b bool) *EscapeLiteralSQL {

	if this.substitute_str_mode == b {
		return this
	}

	this.substitute_str_mode = b
	if this.substitute_str_mode {
		this.substitute_num = "'" + this.substitute + "'"
	} else {
		this.substitute_num = this.substitute
	}
	return this

}

func (this *EscapeLiteralSQL) SetIncludeComment(b bool) *EscapeLiteralSQL {
	this.inclueComment = b
	return this
}

func (this *EscapeLiteralSQL) Process() *EscapeLiteralSQL {

	this.status = NORMAL
	//charsLen := len(chars)

	for this.pos = 0; this.pos < this.length; this.pos++ {
		//	for pos1, ch := range this.sql {
		//		this.pos = pos1
		//		logutil.Printf("Process number=>%s , status = %d", string(this.sql[this.pos]), this.status)

		switch this.sql[this.pos] {
		case '0':
			fallthrough
		case '1':
			fallthrough
		case '2':
			fallthrough
		case '3':
			fallthrough
		case '4':
			fallthrough
		case '5':
			fallthrough
		case '6':
			fallthrough
		case '7':
			fallthrough
		case '8':
			fallthrough
		case '9':
			this._number()
		case ':':
			this._colon()
		case '.':
			this._dot()
		case '-':
			this._minus()
		case '/':
			this._slash()
		case '*':
			this._astar()
		case '\'':
			this._qutation()
		case '\\':
			// TODO python 에서 컬럼명, 테이블 명에 쌍따옴표(") 가 붙는 경우 있음. 추후에 일반화 로직 변경
			// 현재는 PHP 만 적용
			if config.GetConfig().AppType == lang.APP_TYPE_PHP || config.GetConfig().AppType == lang.APP_TYPE_BSM_PHP {
				this._backslash()
			} else {
				this._others()
			}
		case '"':
			// TODO python 에서 컬럼명, 테이블 명에 쌍따옴표(") 가 붙는 경우 있음. 추후에 일반화 로직 변경
			// 현재는 PHP 만 적용
			if config.GetConfig().AppType == lang.APP_TYPE_PHP || config.GetConfig().AppType == lang.APP_TYPE_BSM_PHP {
				this._dqutation()
			} else {
				this._others()
			}
		default:
			this._others()
		}

		//logutil.Printf("Process parsedSql=>%s",this.parsedSql.ToString())
		//logutil.Printf("Process param=>%s",this.param.ToString())

	}

	//	for this.pos = 0; this.pos < this.length; this.pos++ {
	//
	//		switch this.sql[this.pos] {
	//			case '0': fallthrough
	//			case '1': fallthrough
	//			case '2': fallthrough
	//			case '3': fallthrough
	//			case '4': fallthrough
	//			case '5': fallthrough
	//			case '6': fallthrough
	//			case '7': fallthrough
	//			case '8': fallthrough
	//			case '9': fallthrough
	//				this._number()
	//			case ':':
	//				this._colon()
	//			case '.':
	//				this._dot()
	//			case '-':
	//				this._minus()
	//			case '/':
	//				this._slash()
	//			case '*':
	//				this._astar()
	//			case '\'':
	//				this._qutation()
	//			default:
	//				this._others()
	//		}
	//	}

	return this

}

func (this *EscapeLiteralSQL) _others() {
	//fmt.Println("other=>'",string(this.sql[this.pos]),"' ", this.sql[this.pos], "status=" ,this.status)
	switch this.status {
	case COMMENT:
		if this.inclueComment {
			this.parsedSql.WriteByte(this.sql[this.pos])
		}

	case ALPABET:
		this.parsedSql.WriteByte(this.sql[this.pos])
		if isProgLetter(this.sql[this.pos]) == false {
			this.status = NORMAL
		}

	case NUMBER:
		this.parsedSql.WriteByte(this.sql[this.pos])
		this.status = NORMAL

	case QUTATION, DQUTATION:
		this.param.WriteByte(this.sql[this.pos])
	default:
		if isProgLetter(this.sql[this.pos]) {
			this.status = ALPABET
			if this.SqlType == 0 {
				this.define_crud()
			}
		} else {
			this.status = NORMAL
		}
		this.parsedSql.WriteByte(this.sql[this.pos])
	}
}

func isProgLetter(ch uint8) bool {
	//Java return Character.isLetter(ch) || ch == '_';
	return ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ch == '_'
}

func (this *EscapeLiteralSQL) define_crud() {
	this.SqlType = strings.ToUpper(string(this.sql[this.pos]))[0]
	switch this.SqlType {
	case 'S':
		fallthrough
	case 'U':
		fallthrough
	case 'D':
		fallthrough
	case 'I':
	default:
		this.SqlType = '*'
	}
}

func (this *EscapeLiteralSQL) _colon() {
	switch this.status {
	case COMMENT:
		if this.inclueComment {
			this.parsedSql.WriteByte(this.sql[this.pos])
		}
	case QUTATION, DQUTATION:
		this.param.WriteByte(this.sql[this.pos])
	default:
		this.parsedSql.WriteByte(this.sql[this.pos])
		this.status = COLON
	}
}

func (this *EscapeLiteralSQL) _qutation() {
	//logutil.Printf("_qutation = %s, status =%d", string(this.sql[this.pos]), this.status)
	switch this.status {
	case NORMAL:
		if len(this.param.String()) > 0 {
			this.param.WriteByte(byte(','))
		}
		this.param.WriteByte(this.sql[this.pos])
		this.status = QUTATION

	case COMMENT:
		if this.inclueComment {
			this.parsedSql.WriteByte(this.sql[this.pos])
		}
	case ALPABET:
		this.parsedSql.WriteByte(this.sql[this.pos])
		this.status = QUTATION

	case NUMBER:
		this.parsedSql.WriteByte(this.sql[this.pos])
		this.status = QUTATION

	case QUTATION:
		// 따옴표가 두개 연속 오는 경우 종료가 아닌 일반 param으로 따옴표 한개로 추가
		if this.getNext(this.pos) == '\'' {
			this.param.WriteByte(byte('\''))
			this.param.WriteByte(byte('\''))
			this.pos++
			return
		}
		this.param.WriteByte(byte('\''))
		//logutil.Printf("_qutation:QUATAION:param=%s", this.param.ToString())
		this.parsedSql.WriteByte(byte('\''))
		this.parsedSql.Write([]byte(this.substitute))
		this.parsedSql.WriteByte(byte('\''))

		//logutil.Printf("_qutation:QUATAION:parsedSql=%s", this.parsedSql.ToString())
		this.status = NORMAL

	case DQUTATION:
		this.param.WriteByte(this.sql[this.pos])

	}
}

// 20170925 쌍 따옴표 (double quataion) 추가
func (this *EscapeLiteralSQL) _dqutation() {
	//logutil.Printf("_qutation = %s, status =%d", string(this.sql[this.pos]), this.status)
	switch this.status {
	case NORMAL:
		if len(this.param.String()) > 0 {
			this.param.WriteByte(byte(','))
		}
		this.param.WriteByte(this.sql[this.pos])
		this.status = DQUTATION

	case COMMENT:
		if this.inclueComment {
			this.parsedSql.WriteByte(this.sql[this.pos])
		}

	case ALPABET:
		this.parsedSql.WriteByte(this.sql[this.pos])
		this.status = DQUTATION

	case NUMBER:
		this.parsedSql.WriteByte(this.sql[this.pos])
		this.status = DQUTATION

	case QUTATION:
		this.param.WriteByte(this.sql[this.pos])

	case DQUTATION:
		this.param.WriteByte(byte('"'))
		//logutil.Printf("_qutation:QUATAION:param=%s", this.param.ToString())
		//this.parsedSql.Append("'").Append(this.substitute).Append("'")
		this.parsedSql.Write([]byte(this.substitute))
		//logutil.Printf("_qutation:QUATAION:parsedSql=%s", this.parsedSql.ToString())
		this.status = NORMAL
	}
}

func (this *EscapeLiteralSQL) _astar() {
	switch this.status {
	case COMMENT:
		if this.inclueComment {
			this.parsedSql.WriteByte(this.sql[this.pos])
		}
		if this.getNext(this.pos) == '/' {
			if this.inclueComment {
				this.parsedSql.WriteByte(byte('/'))
			}
			this.pos++
			this.status = NORMAL
		}

	case QUTATION, DQUTATION:
		this.param.WriteByte(this.sql[this.pos])

	default:
		this.parsedSql.WriteByte(this.sql[this.pos])
		this.status = NORMAL
	}
}

func (this *EscapeLiteralSQL) _slash() {
	switch this.status {
	case COMMENT:
		if this.inclueComment {
			this.parsedSql.WriteByte(this.sql[this.pos])
		}
	case QUTATION, DQUTATION:
		this.param.WriteByte(this.sql[this.pos])

	default:
		if this.getNext(this.pos) == '*' {
			if this.inclueComment {
				this.parsedSql.WriteByte(this.sql[this.pos])
				this.parsedSql.WriteByte(byte('*'))
			}
			this.pos++
			this.status = COMMENT
		}
	}
}

func (this *EscapeLiteralSQL) _backslash() {
	switch this.status {
	case COMMENT:
		if this.inclueComment {
			this.parsedSql.WriteByte(this.sql[this.pos])
		}
	case QUTATION, DQUTATION:
		if this.getNext(this.pos) != 0 {
			this.pos++
			this.param.WriteByte(byte('\\'))
			this.param.WriteByte(this.sql[this.pos])
		}

		//this.param.Append(string(this.sql[this.pos]))
	default:
		if this.getNext(this.pos) != 0 {

			switch this.sql[this.pos+1] {
			// 'update .... ast=\'aaa\' ;"
			case '\'':
				this.pos++
				this.param.WriteByte(byte('\''))
				this.status = QUTATION
			// "update .... ast=\"aaa\" ;"
			case '"':
				this.pos++
				this.param.WriteByte(byte('"'))
				this.status = DQUTATION
			default:
				this.pos++
				this.parsedSql.WriteByte(this.sql[this.pos])
			}
		}
	}
}

func (this *EscapeLiteralSQL) _minus() {
	switch this.status {
	case COMMENT:
		if this.inclueComment {
			this.parsedSql.WriteByte(this.sql[this.pos])
		}
	case QUTATION, DQUTATION:
		this.param.WriteByte(this.sql[this.pos])
	default:
		if this.getNext(this.pos) == '-' {
			if this.inclueComment {
				this.parsedSql.WriteByte(this.sql[this.pos])
			}
			for this.sql[this.pos] != '\n' {
				this.pos++
				if this.pos < this.length {
					if this.inclueComment {
						this.parsedSql.WriteByte(this.sql[this.pos])
					}
				} else {
					break
				}
			}
		} else {
			this.parsedSql.WriteByte(this.sql[this.pos])
		}
		this.status = NORMAL
	}
}
func (this *EscapeLiteralSQL) _dot() {
	switch this.status {
	case NORMAL:
		this.parsedSql.WriteByte(this.sql[this.pos])
	case COMMENT:
		if this.inclueComment {
			this.parsedSql.WriteByte(this.sql[this.pos])
		}
	case ALPABET:
		this.parsedSql.WriteByte(this.sql[this.pos])
		this.status = NORMAL

	case NUMBER:
		this.param.WriteByte(this.sql[this.pos])
	case QUTATION, DQUTATION:
		this.param.WriteByte(this.sql[this.pos])
	}
}
func (this *EscapeLiteralSQL) _number() {
	switch this.status {
	case NORMAL:
		if len(this.param.String()) > 0 {
			this.param.WriteByte(byte(','))
		}
		this.param.WriteByte(this.sql[this.pos])
		this.parsedSql.Write([]byte(this.substitute_num))
		this.status = NUMBER

	case COMMENT:
		if this.inclueComment {
			this.parsedSql.WriteByte(this.sql[this.pos])
		}
	case COLON:
		fallthrough
	case ALPABET:
		this.parsedSql.WriteByte(this.sql[this.pos])

	case NUMBER:
		fallthrough
	case QUTATION, DQUTATION:
		this.param.WriteByte(this.sql[this.pos])

	}
}
func (this *EscapeLiteralSQL) getNext(x int) uint8 {
	if x < this.length-1 {
		return this.sql[x+1]
	} else {
		return 0
	}
	//return x < length ? this.sql[x + 1] : 0

}

func (this *EscapeLiteralSQL) GetParsedSql() string {
	return this.parsedSql.String()
}
func (this *EscapeLiteralSQL) GetParameter() string {
	return this.param.String()
}

//func main() {
//	s := `"/*
//211.250.217.119 <br>
//2048609494 <br>
///webservice_root/pravs/shop/admin_provider/order/indb.php (line:235) <br>
///webservice_root/pravs/shop/lib/db.class.php (line:130)
//*/"
//	select name from gd_member
//	-- aaaaalkjl
//	where m_no='$'
///*
//211.250.217.119 <br>
//2048609494 <br>
///webservice_root/pravs/shop/admin_provider/order/indb.php (line:235) <br>
///webservice_root/pravs/shop/lib/db.class.php (line:130)
//*/"`
//	time := dateutil.Now()
//	ec := NewEscapeLiteralSQL(s)
//	ec.SetIncludeComment(false)
//	ec.Process()
//
//	etime := dateutil.Now()
//	//FileUtil.save("d:/tmp/sample-query2.out",ec.parsedSql.toString().getBytes())
//	fmt.Println("SQL: ", ec.GetParsedSql(), " ", (etime - time), " ms")
//	fmt.Println("PARAM: ", ec.GetParameter())
//	fmt.Println("type: ", ec.SqlType)
//
//	//fmt.Println("‘".getBytes()[0] & 0xff)
//}
