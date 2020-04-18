/*
   Copyright 2020 Markus Hinz

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

package provider

import (
	"io"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3/s3manager/s3manageriface"
	"github.com/stretchr/testify/assert"
)

type mockItem struct {
	key   string
	size  int64
	bytes []byte
}

type mockClient struct {
	s3iface.S3API
	items []mockItem
	err   error
}

func (mock mockClient) ListObjectsV2(input *s3.ListObjectsV2Input) (output *s3.ListObjectsV2Output, err error) {
	var contents []*s3.Object
	for _, item := range mock.items {
		key := item.key
		size := item.size
		contents = append(contents, &s3.Object{
			Key:  &key,
			Size: &size,
		})
	}
	return &s3.ListObjectsV2Output{Contents: contents}, mock.err
}

type mockDownloader struct {
	s3manageriface.DownloaderAPI
	mockItem mockItem
	err      error
}

func (mock mockDownloader) Download(writer io.WriterAt, input *s3.GetObjectInput, options ...func(*s3manager.Downloader)) (size int64, err error) {
	writer.WriteAt(mock.mockItem.bytes, 0)
	return int64(len(mock.mockItem.bytes)), mock.err
}

func TestListEmails(t *testing.T) {
	type args struct {
		provider   awsS3Provider
		notNumbers []int
	}
	tests := []struct {
		name    string
		args    args
		want    map[int]*email
		wantErr bool
	}{
		{
			name: "no emails",
			args: args{
				provider: awsS3Provider{
					client: mockClient{},
				},
			},
		},
		{
			name: "no emails excluded",
			args: args{
				provider: awsS3Provider{
					client: mockClient{},
				},
				notNumbers: []int{2},
			},
		},
		{
			name: "emails",
			args: args{
				provider: awsS3Provider{
					client: mockClient{
						items: []mockItem{
							{
								key:  "abc123",
								size: 1000,
							},
							{
								key:  "def456",
								size: 2000,
							},
							{
								key:  "ghi789",
								size: 3000,
							},
						},
					},
				},
			},
			want: map[int]*email{
				1: {
					ID:   "abc123",
					Size: 1000,
				},
				2: {
					ID:   "def456",
					Size: 2000,
				},
				3: {
					ID:   "ghi789",
					Size: 3000,
				},
			},
		},
		{
			name: "emails excluded",
			args: args{
				provider: awsS3Provider{
					client: mockClient{
						items: []mockItem{
							{
								key:  "abc123",
								size: 1000,
							},
							{
								key:  "def456",
								size: 2000,
							},
							{
								key:  "ghi789",
								size: 3000,
							},
						},
					},
				},
				notNumbers: []int{-10, 2, 7},
			},
			want: map[int]*email{
				1: {
					ID:   "abc123",
					Size: 1000,
				},
				3: {
					ID:   "ghi789",
					Size: 3000,
				},
			},
		},
		{
			name: "cache",
			args: args{
				provider: awsS3Provider{
					client: mockClient{
						items: []mockItem{
							{
								key:  "abc123",
								size: 1000,
							},
							{
								key:  "def456",
								size: 2000,
							},
							{
								key:  "shouldNotBeLoaded",
								size: 0000,
							},
						},
					},
					cache: &awsS3Cache{
						emails: map[int]*email{
							1: {
								ID:   "abc123",
								Size: 1000,
							},
							2: {
								ID:   "def456",
								Size: 2000,
							},
							3: {
								ID:   "ghi789",
								Size: 3000,
							},
						},
					},
				},
				notNumbers: []int{2},
			},
			want: map[int]*email{
				1: {
					ID:   "abc123",
					Size: 1000,
				},
				3: {
					ID:   "ghi789",
					Size: 3000,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.args.provider.ListEmails(tt.args.notNumbers)
			assert.EqualValues(t, tt.wantErr, err != nil)
			assert.EqualValues(t, len(tt.want), len(got))
			for id, email := range got {
				assert.Contains(t, tt.want, id)
				assert.EqualValues(t, tt.want[id], email)
			}
		})
	}
}
