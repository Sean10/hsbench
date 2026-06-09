package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"sync/atomic"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// verifyObjectData 比较 data 是否与 generateObjectData(objnum, key, len(data)) 相同。
// 用于 GET 验证和 inline 验证。
func verifyObjectData(objnum int64, key uint8, data []byte) bool {
	expected := generateObjectData(objnum, key, len(data))
	return bytes.Equal(expected, data)
}

// buildVerifyReport 构造数据不一致时的错误日志字符串，包含 objnum/bucket/key/hex 前缀。
// actual 最多取前 64 字节做 hex 展示。
func buildVerifyReport(objnum int64, bucket string, key uint8, actual []byte) string {
	limit := 64
	if len(actual) < limit {
		limit = len(actual)
	}
	hexStr := hex.EncodeToString(actual[:limit])
	return fmt.Sprintf("VERIFY MISMATCH: objnum=%d bucket=%s key=%d actual_hex[0:64]=%s",
		objnum, bucket, key, hexStr)
}

// classifyKeyByte 将 KeyMap 中的单字节值分类为 "unwritten"、"written"、"busy" 或 "dv_error"。
func classifyKeyByte(b uint8) string {
	switch {
	case b&0x80 != 0:
		return "busy"
	case b == dvError:
		return "dv_error"
	case b == 0:
		return "unwritten"
	default:
		return "written"
	}
}

// buildVerifySummary 构造 v 模式完成后的汇总字符串。
func buildVerifySummary(total, verified, pass, fail, unknown, dvErrorCount, unwritten int64) string {
	return fmt.Sprintf(
		"VERIFY SUMMARY: total=%d verified=%d pass=%d fail=%d unknown=%d dv_error=%d unwritten=%d",
		total, verified, pass, fail, unknown, dvErrorCount, unwritten)
}

// runVerify uses verifyObjCounter (independent of op_counter) to partition objects
// across threads, always scanning from object 0 regardless of -f (first_object).
// Each goroutine claims the next object atomically. Counters are accumulated into
// the global verify* atomics; the summary is printed once by runWrapper after all
// threads have finished.
func runVerify(thread_num int, svc *s3.S3, stats *Stats) {
	if globalKeyMap == nil {
		log.Printf("runVerify: no key_map loaded, skipping verification")
		stats.finish(thread_num)
		atomic.AddInt64(&running_threads, -1)
		return
	}

	total := int64(len(globalKeyMap.data))

	for {
		objnum := atomic.AddInt64(&verifyObjCounter, 1)
		if objnum >= total {
			break
		}

		b := globalKeyMap.ReadKey(objnum)
		class := classifyKeyByte(b)
		atomic.AddInt64(&verifyTotal, 1)

		switch class {
		case "unwritten":
			atomic.AddInt64(&verifyUnwritten, 1)
			continue
		case "busy":
			atomic.AddInt64(&verifyUnknown, 1)
			continue
		case "dv_error":
			atomic.AddInt64(&verifyDVError, 1)
			continue
		}

		// written: GET and verify
		key := b & 0x7F
		bucket_num := objnum % int64(bucket_count)
		objKey := fmt.Sprintf("%s%012d", object_prefix, objnum)

		r := &s3.GetObjectInput{
			Bucket: aws.String(buckets[bucket_num]),
			Key:    aws.String(objKey),
		}
		req, resp := svc.GetObjectRequest(r)
		err := req.Send()
		if err != nil {
			log.Printf("runVerify: GET failed objnum=%d: %v", objnum, err)
			atomic.AddInt64(&verifyFail, 1)
			continue
		}

		data, readErr := readBodyFull(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			log.Printf("runVerify: read body failed objnum=%d: %v", objnum, readErr)
			atomic.AddInt64(&verifyFail, 1)
			continue
		}

		if verifyObjectData(objnum, key, data) {
			atomic.AddInt64(&verifyPass, 1)
		} else {
			atomic.AddInt64(&verifyFail, 1)
			log.Printf("%s", buildVerifyReport(objnum, buckets[bucket_num], key, data))
		}
	}

	stats.finish(thread_num)
	atomic.AddInt64(&running_threads, -1)
}

