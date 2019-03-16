/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fsblkstorage

////  WRITER ////
type blockfileWriter struct {
	filePath string
}

func newBlockfileWriter(filePath string) (*blockfileWriter, error) {
	writer := &blockfileWriter{filePath: filePath}
	return writer, writer.open()
}

func (w *blockfileWriter) truncateFile(targetSize int) error {

	return nil
}

func (w *blockfileWriter) append(b []byte, sync bool) error {
	return nil
}

func (w *blockfileWriter) open() error {
	return nil
}

func (w *blockfileWriter) close() error {
	return nil
}

////  READER ////
type blockfileReader struct {
}

func newBlockfileReader(filePath string) (*blockfileReader, error) {
	reader := &blockfileReader{}
	return reader, nil
}

func (r *blockfileReader) read(offset int, length int) ([]byte, error) {
	b := make([]byte, length)
	return b, nil
}

func (r *blockfileReader) close() error {
	return nil
}
