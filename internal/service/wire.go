package service

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// magic is the word every request starts with.
const magic = "ACER"

// Request is a request of the protocol.
type Request struct {
	// Packet is the identifier of the kind of the request.
	Packet uint32
	// Function is the name of the setting the request is about.
	Function string
	// Parameter is the value that is written, it is empty for a request
	// that only reads.
	Parameter any
}

// readRequest reads one request from a stream.
//
// The word, the identifier and the document are sent in one piece, so the
// reader waits for the whole of them.
func readRequest(reader io.Reader) (Request, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(reader, header); err != nil {
		return Request{}, err
	}
	if string(header[:4]) != magic {
		return Request{}, fmt.Errorf("service: 请求没有以 %s 开头", magic)
	}

	request := Request{Packet: binary.LittleEndian.Uint32(header[4:])}

	// The document behind the header is a JSON text, its end is found by
	// counting the braces.
	body, err := readJSON(reader)
	if err != nil {
		return Request{}, err
	}
	if len(body) == 0 {
		return request, nil
	}

	var payload struct {
		Function  string `json:"Function"`
		Parameter any    `json:"Parameter"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Request{}, fmt.Errorf("service: 请求内容无法解析: %w", err)
	}
	request.Function = payload.Function
	request.Parameter = payload.Parameter
	return request, nil
}

// readJSON reads a JSON document from a stream.
//
// The document is an object, so the reader counts the braces and stops when
// they are balanced again.
func readJSON(reader io.Reader) ([]byte, error) {
	out := make([]byte, 0, 256)
	depth := 0
	started := false
	buffer := make([]byte, 1)

	for {
		_, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF && started {
				return out, nil
			}
			return out, err
		}
		out = append(out, buffer[0])

		switch buffer[0] {
		case '{':
			depth++
			started = true
		case '}':
			depth--
			if started && depth == 0 {
				return out, nil
			}
		}
	}
}

// writeResponse writes one answer.
//
// The answer is a JSON document on its own, the shape of it is the one the
// official application expects.
func writeResponse(writer io.Writer, function string, data any) error {
	document := map[string]any{
		"data":    data,
		"request": function,
		"result":  "0",
	}
	text := writeJSON(document)
	if _, err := writer.Write([]byte(text)); err != nil {
		return err
	}
	return nil
}

// writeSetResponse writes the answer to a request that writes a setting.
//
// The original service leaves the data out of this answer, it only tells
// whether the change worked.
func writeSetResponse(writer io.Writer, function string) error {
	document := map[string]any{
		"request": function,
		"result":  "0",
	}
	_, err := writer.Write([]byte(writeJSON(document)))
	return err
}

// writeSetError writes the answer to a request that could not be carried
// out.
func writeSetError(writer io.Writer, function string, code string) error {
	document := map[string]any{
		"request": function,
		"result":  code,
	}
	_, err := writer.Write([]byte(writeJSON(document)))
	return err
}

// writeUnsupportedResponse writes the answer to a request that the
// original service does not implement.
//
// The answer carries no data field at all, only the name of the request
// and the code of the failure. The official application reads the code and
// stops there, it never looks for the data.
func writeUnsupportedResponse(writer io.Writer, function string) error {
	document := map[string]any{
		"request": function,
		"result":  "3",
	}
	_, err := writer.Write([]byte(writeJSON(document)))
	return err
}

// writeEmptyResponse writes an answer without a name.
//
// The original service answers in this way when it does not know the
// request, and the official application expects it.
func writeEmptyResponse(writer io.Writer) error {
	document := map[string]any{
		"data":    nil,
		"request": nil,
		"result":  "0",
	}
	_, err := writer.Write([]byte(writeJSON(document)))
	return err
}

// writeError writes an answer that carries an error.
func writeError(writer io.Writer, function string, code string) error {
	document := map[string]any{
		"data":    nil,
		"request": function,
		"result":  code,
	}
	_, err := writer.Write([]byte(writeJSON(document)))
	return err
}
