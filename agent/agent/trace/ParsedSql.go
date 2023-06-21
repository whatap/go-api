package trace

import (
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/stringutil"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/util/sqlutil"
)

type ParsedSql struct {
	Type  byte
	Sql   int32
	Param string
}

func NewParsedSql(Type byte, Sql int32, Param string) *ParsedSql {
	p := new(ParsedSql)

	p.Type = Type
	p.Sql = Sql
	p.Param = Param

	return p
}

// Java TraceSQL
//var checkedSql *hmap.IntKeyLinkedMap = hmap.NewIntKeyLinkedMapDefault().SetMax(1001)
//var nonLiteSql *hmap.IntKeyLinkedMap = hmap.NewIntKeyLinkedMapDefault().SetMax(10000)

//var checkedSql *hmap.IntKeyLinkedMap = hmap.NewIntKeyLinkedMapDefault().SetMax(1001)
//var nonLiteSql *hmap.IntKeyLinkedMap = hmap.NewIntKeyLinkedMapDefault().SetMax(10000)

var checkedSql *hmap.IntKeyLinkedMap = hmap.NewIntKeyLinkedMap(10001, 1.0).SetMax(1001)
var nonLiteSql *hmap.IntKeyLinkedMap = hmap.NewIntKeyLinkedMap(1007, 1.0).SetMax(10000)

func resetSqlText() {
	checkedSql.Clear()
	nonLiteSql.Clear()
}

func EscapeLiteral(sql string) *ParsedSql {
	conf := config.GetConfig()
	if !conf.TraceSqlNormalizeEnabled {
		hash := hash.HashStr(sql)
		data.SendHashText(pack.TEXT_SQL, hash, sql)
		return NewParsedSql('*', hash, "")
	}

	sqlHash := int32(stringutil.HashCode(sql))

	var psql *ParsedSql
	if psql, _ := nonLiteSql.Get(sqlHash).(*ParsedSql); psql != nil {
		return psql
	}
	if psql, _ := checkedSql.Get(sqlHash).(*ParsedSql); psql != nil {
		return psql
	}

	els := sqlutil.NewEscapeLiteralSQL(sql)
	els.SetIncludeComment(conf.ProfileSqlCommentEnabled)
	els.Process()

	sql2 := els.GetParsedSql()
	hash := hash.HashStr(sql2)
	data.SendHashText(pack.TEXT_SQL, hash, sql2)
	if hash == sqlHash {
		psql = NewParsedSql(els.SqlType, hash, "")
		nonLiteSql.Put(sqlHash, psql)
	} else {
		psql = NewParsedSql(els.SqlType, hash, els.GetParameter())
		checkedSql.Put(sqlHash, psql)
	}
	return psql
}
