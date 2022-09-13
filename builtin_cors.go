package pi

import "net/http"

func Cors(next HandlerFunc) HandlerFunc {
	return func(ctx Context) error {
		ctx.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.Header().Set("Access-Control-Allow-Credentials", "true")

		if ctx.Is(http.MethodOptions) {
			ctx.Header().Set("Access-Control-Methods", "POST, PUT, PATCH, DELETE")
			ctx.Header().Set("Access-Control-Allow-Headers", "*")
			ctx.Header().Set("Access-Control-Max-Age", "86400")
			ctx.WriteHeader(http.StatusNoContent)
			return nil
		}

		return next(ctx)
	}
}
