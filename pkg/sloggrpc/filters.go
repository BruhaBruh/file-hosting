package sloggrpc

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Filter func(ctx context.Context, info *grpc.UnaryServerInfo) bool

// Basic
func Accept(filter Filter) Filter { return filter }
func Ignore(filter Filter) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		return !filter(ctx, info)
	}
}

// Method (gRPC method name)
func AcceptMethod(methods ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		method := parseMethod(info.FullMethod)

		for _, m := range methods {
			if m == method {
				return true
			}
		}

		return false
	}
}

func IgnoreMethod(methods ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		method := parseMethod(info.FullMethod)

		for _, m := range methods {
			if m == method {
				return false
			}
		}

		return true
	}
}

// Full Method (full gRPC method path like /package.Service/Method)
func AcceptFullMethod(fullMethods ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		return slices.Contains(fullMethods, info.FullMethod)
	}
}

func IgnoreFullMethod(fullMethods ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		return !slices.Contains(fullMethods, info.FullMethod)
	}
}

func AcceptFullMethodContains(parts ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		for _, part := range parts {
			if strings.Contains(info.FullMethod, part) {
				return true
			}
		}

		return false
	}
}

func IgnoreFullMethodContains(parts ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		for _, part := range parts {
			if strings.Contains(info.FullMethod, part) {
				return false
			}
		}

		return true
	}
}

func AcceptFullMethodPrefix(prefixes ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(info.FullMethod, prefix) {
				return true
			}
		}

		return false
	}
}

func IgnoreFullMethodPrefix(prefixes ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(info.FullMethod, prefix) {
				return false
			}
		}

		return true
	}
}

func AcceptFullMethodSuffix(suffixes ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		for _, suffix := range suffixes {
			if strings.HasSuffix(info.FullMethod, suffix) {
				return true
			}
		}

		return false
	}
}

func IgnoreFullMethodSuffix(suffixes ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		for _, suffix := range suffixes {
			if strings.HasSuffix(info.FullMethod, suffix) {
				return false
			}
		}

		return true
	}
}

func AcceptFullMethodMatch(regs ...regexp.Regexp) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		for _, reg := range regs {
			if reg.MatchString(info.FullMethod) {
				return true
			}
		}

		return false
	}
}

func IgnoreFullMethodMatch(regs ...regexp.Regexp) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		for _, reg := range regs {
			if reg.MatchString(info.FullMethod) {
				return false
			}
		}

		return true
	}
}

// Service (package.Service part of /package.Service/Method)
func AcceptService(services ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		service := parseService(info.FullMethod)

		for _, s := range services {
			if s == service {
				return true
			}
		}

		return false
	}
}

func IgnoreService(services ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		service := parseService(info.FullMethod)

		for _, s := range services {
			if s == service {
				return false
			}
		}

		return true
	}
}

func AcceptServicePrefix(prefixes ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		service := parseService(info.FullMethod)

		for _, prefix := range prefixes {
			if strings.HasPrefix(service, prefix) {
				return true
			}
		}

		return false
	}
}

func IgnoreServicePrefix(prefixes ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		service := parseService(info.FullMethod)

		for _, prefix := range prefixes {
			if strings.HasPrefix(service, prefix) {
				return false
			}
		}

		return true
	}
}

// Status Code (gRPC codes)
func AcceptStatus(statuses ...codes.Code) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		// Get status from context if available
		if err := ctx.Err(); err != nil {
			st, _ := status.FromError(err)
			return slices.Contains(statuses, st.Code())
		}
		return false
	}
}

func IgnoreStatus(statuses ...codes.Code) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		if err := ctx.Err(); err != nil {
			st, _ := status.FromError(err)
			return !slices.Contains(statuses, st.Code())
		}
		return true
	}
}

func AcceptStatusOK() Filter {
	return AcceptStatus(codes.OK)
}

func IgnoreStatusOK() Filter {
	return IgnoreStatus(codes.OK)
}

func AcceptClientErrors() Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		if err := ctx.Err(); err != nil {
			st, _ := status.FromError(err)
			return isClientError(st.Code())
		}
		return false
	}
}

func IgnoreClientErrors() Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		if err := ctx.Err(); err != nil {
			st, _ := status.FromError(err)
			return !isClientError(st.Code())
		}
		return true
	}
}

func AcceptServerErrors() Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		if err := ctx.Err(); err != nil {
			st, _ := status.FromError(err)
			return isServerError(st.Code())
		}
		return false
	}
}

func IgnoreServerErrors() Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		if err := ctx.Err(); err != nil {
			st, _ := status.FromError(err)
			return !isServerError(st.Code())
		}
		return true
	}
}

// Metadata
func AcceptMetadata(key string, values ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return false
		}

		mdValues := md.Get(key)
		if len(mdValues) == 0 {
			return false
		}

		for _, value := range values {
			if slices.Contains(mdValues, value) {
				return true
			}
		}

		return false
	}
}

func IgnoreMetadata(key string, values ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return true
		}

		mdValues := md.Get(key)
		if len(mdValues) == 0 {
			return true
		}

		for _, value := range values {
			if slices.Contains(mdValues, value) {
				return false
			}
		}

		return true
	}
}

func AcceptMetadataExists(keys ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return false
		}

		for _, key := range keys {
			if len(md.Get(key)) > 0 {
				return true
			}
		}

		return false
	}
}

func IgnoreMetadataExists(keys ...string) Filter {
	return func(ctx context.Context, info *grpc.UnaryServerInfo) bool {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return true
		}

		for _, key := range keys {
			if len(md.Get(key)) > 0 {
				return false
			}
		}

		return true
	}
}

// Helper function to parse service name from full method
func parseService(fullMethod string) string {
	// fullMethod format: /package.Service/Method
	// Extract package.Service part
	if len(fullMethod) > 0 && fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}

	parts := strings.Split(fullMethod, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	return fullMethod
}
