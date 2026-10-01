# Downloads a file and checks its SHA-256 (build time only).
import hashlib
import sys
import urllib.request

url, dst, want = sys.argv[1:4]
data = urllib.request.urlopen(url, timeout=600).read()
got = hashlib.sha256(data).hexdigest()
if got != want:
    sys.exit(f"checksum mismatch for {url}: {got}")
open(dst, "wb").write(data)
