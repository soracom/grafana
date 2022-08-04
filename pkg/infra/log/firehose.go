package log

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	gokitlog "github.com/go-kit/log"
)

const (
	maxBatchRecords = 500
)

type TimestampLogHandler struct {
	Handler gokitlog.Logger
}

const (
	httpClientDefaultTimeoutDuration = 10 * time.Second
)

var httpClient = &http.Client{
	Timeout: httpClientDefaultTimeoutDuration,
}

func NewTimestampLogHandlerWithHandler(handler gokitlog.Logger) *TimestampLogHandler {
	return &TimestampLogHandler{Handler: handler}
}

// Log is a standard log but changes the keyname for time to "timestamp"
func (fhlh *TimestampLogHandler) Log(keyvals ...interface{}) error {
	// TODO iterate through event offsets looking for "Time" and change to "timestamp"
	//r.KeyNames.Time = "timestamp"
	return fhlh.Handler.Log(keyvals...)
}

// FirehoseWriter is a general purpose writer for AWS Firehose.
// Amazon Kinesis Firehose is a fully-managed service that delivers real-time
// streaming data to destinations such as Amazon Simple Storage Service (Amazon
// S3), Amazon Elasticsearch Service (Amazon ES), and Amazon Redshift.
type FirehoseWriter struct {
	client     *firehose.Firehose
	buf        [][]byte
	bufCh      chan []byte
	flushCh    chan bool
	errCh      chan error
	streamName string
}

func (fw *FirehoseWriter) Write(b []byte) (int, error) {
	// New log system seems to overwrite existing slices?
	// Create a copy so that wont happen while waiting for the channel
	// to be read
	slicecopy := append([]byte(nil), b...)
	fw.bufCh <- slicecopy
	return len(b), nil
}

// NewFirehose returns initialized FirehoseWriter with persistent Firehose logger.
func NewFirehose(streamName string, awsRegion string) (*FirehoseWriter, error) {

	conf := &aws.Config{
		Region:     aws.String(awsRegion),
		HTTPClient: httpClient,
	}
	sess, err := session.NewSession(conf)
	if err != nil {
		return nil, err
	}

	svc := firehose.New(sess)

	bufCh := make(chan []byte, 1000)
	flushCh := make(chan bool)
	errCh := make(chan error)

	fw := &FirehoseWriter{
		client:     svc,
		buf:        make([][]byte, 0),
		bufCh:      bufCh,
		flushCh:    flushCh,
		errCh:      errCh,
		streamName: streamName,
	}

	go fw.intervalLoop()
	go fw.bufLoop()

	return fw, nil
}

func (fw *FirehoseWriter) intervalLoop() {
	for {
		time.Sleep(15 * time.Second)
		fw.flushCh <- true
	}
}

func (fw *FirehoseWriter) bufLoop() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintf(os.Stderr, "panic: %+v\n", err)
		}
	}()
	for {
		select {
		case e := <-fw.bufCh:
			fw.buf = append(fw.buf, e)
		case <-fw.flushCh:
			fw.flush()
		}
	}
}

func (fw *FirehoseWriter) flush() {
	if len(fw.buf) == 0 {
		return
	}

	defer func() {
		fw.buf = make([][]byte, 0)
	}()

	for _, buf := range splitBuf(fw.buf, maxBatchRecords) {
		records := make([]*firehose.Record, 0, len(buf))
		for _, e := range buf {
			records = append(records, &firehose.Record{
				Data: e,
			})
		}

		in := &firehose.PutRecordBatchInput{
			DeliveryStreamName: aws.String(fw.streamName),
			Records:            records,
		}

		_, err := fw.client.PutRecordBatch(in)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			//fw.errCh <- err
		}
	}
}

func splitBuf(buf [][]byte, size int) [][][]byte {
	result := make([][][]byte, 0)
	for len(buf) > 0 {
		if len(buf) > size {
			result = append(result, buf[:size])
			buf = buf[size:]
		} else {
			result = append(result, buf)
			buf = buf[:0]
		}
	}
	return result
}
