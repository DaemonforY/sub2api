
window.__ASSETS = {};
(function () { const cache = {};
  window.__A = function (p) {
    if (cache[p]) return cache[p];
    const d = window.__ASSETS[p]; if (!d) return p;
    const i = d.indexOf(','), bin = atob(d.slice(i + 1)), u = new Uint8Array(bin.length);
    for (let k = 0; k < bin.length; k++) u[k] = bin.charCodeAt(k);
    return (cache[p] = URL.createObjectURL(new Blob([u], { type: d.slice(5, d.indexOf(';')) })));
  };
})();
