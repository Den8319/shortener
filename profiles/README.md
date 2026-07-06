Type: inuse_space
Time: 2026-07-06 19:19:45 MSK
Showing nodes accounting for 8791.87kB, 322.17% of 2728.98kB total
      flat  flat%   sum%        cum   cum%
11520.86kB 422.17% 422.17% 11520.86kB 422.17%  <unknown>
   -1026kB 37.60% 384.57%    -1026kB 37.60%  runtime.allocm
 -678.95kB 24.88% 359.69% -1190.96kB 43.64%  github.com/Den8319/shortener/internal/repository/file.(*Fileloader).GetUserURLs
 -512.02kB 18.76% 340.93%  -512.02kB 18.76%  internal/sync.newEntryNode[go.shape.interface {},go.shape.interface {}] (inline)
 -512.01kB 18.76% 322.17%  -512.01kB 18.76%  encoding/json.(*decodeState).literalStore
         0     0% 322.17%  -512.02kB 18.76%  encoding/json.(*Encoder).Encode
         0     0% 322.17%  -512.01kB 18.76%  encoding/json.(*decodeState).object
         0     0% 322.17%  -512.01kB 18.76%  encoding/json.(*decodeState).unmarshal
         0     0% 322.17%  -512.01kB 18.76%  encoding/json.(*decodeState).value
         0     0% 322.17%  -512.02kB 18.76%  encoding/json.(*encodeState).marshal
         0     0% 322.17%  -512.02kB 18.76%  encoding/json.(*encodeState).reflectValue
         0     0% 322.17%  -512.01kB 18.76%  encoding/json.Unmarshal
         0     0% 322.17%  -512.02kB 18.76%  encoding/json.typeEncoder
         0     0% 322.17%  -512.02kB 18.76%  encoding/json.valueEncoder
         0     0% 322.17% -1702.98kB 62.40%  github.com/Den8319/shortener/internal/auth.WithAuth.func1
         0     0% 322.17% -1702.98kB 62.40%  github.com/Den8319/shortener/internal/compress.WithCompression.func1
         0     0% 322.17% -1702.98kB 62.40%  github.com/Den8319/shortener/internal/handler.(*Handler).GetUserURLsHandler
         0     0% 322.17% -1702.98kB 62.40%  github.com/Den8319/shortener/internal/logger.WithLogging.func1
         0     0% 322.17% -1190.96kB 43.64%  github.com/Den8319/shortener/internal/service/store.(*Store).GetUserURLs
         0     0% 322.17% -1702.98kB 62.40%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0% 322.17% -1702.98kB 62.40%  github.com/go-chi/chi/v5.(*Mux).routeHTTP
         0     0% 322.17%  -512.02kB 18.76%  internal/sync.(*HashTrieMap[go.shape.interface {},go.shape.interface {}]).Store (inline)
         0     0% 322.17%  -512.02kB 18.76%  internal/sync.(*HashTrieMap[go.shape.interface {},go.shape.interface {}]).Swap
         0     0% 322.17%  -512.02kB 18.76%  internal/sync.(*entry[go.shape.interface {},go.shape.interface {}]).swap
         0     0% 322.17% -1702.98kB 62.40%  net/http.(*conn).serve
         0     0% 322.17% -1702.98kB 62.40%  net/http.HandlerFunc.ServeHTTP
         0     0% 322.17% -1702.98kB 62.40%  net/http.serverHandler.ServeHTTP
         0     0% 322.17%     -513kB 18.80%  runtime.mcall
         0     0% 322.17%     -513kB 18.80%  runtime.mstart
         0     0% 322.17%     -513kB 18.80%  runtime.mstart0
         0     0% 322.17%     -513kB 18.80%  runtime.mstart1
         0     0% 322.17%    -1026kB 37.60%  runtime.newm
         0     0% 322.17%     -513kB 18.80%  runtime.park_m
         0     0% 322.17%    -1026kB 37.60%  runtime.resetspinning
         0     0% 322.17%    -1026kB 37.60%  runtime.schedule
         0     0% 322.17%    -1026kB 37.60%  runtime.startm
         0     0% 322.17%    -1026kB 37.60%  runtime.wakep
         0     0% 322.17%  -512.02kB 18.76%  sync.(*Map).Store (inline)
