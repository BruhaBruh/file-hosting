package grpcinterceptors

import "google.golang.org/grpc/metadata"

func extractMetadata(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}
