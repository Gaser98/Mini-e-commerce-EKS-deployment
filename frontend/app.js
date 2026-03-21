const API = window.location.origin;
let token = localStorage.getItem("token");
let cartN = 0;

const META = {
  "Wireless Mouse":       { img: "https://images.pexels.com/photos/17821147/pexels-photo-17821147.jpeg?auto=compress&cs=tinysrgb&w=400", badge: "POPULAR",    bc: "red",   cat: "Peripherals",  r: "4.8", rv: "234", old: "39.99" },
  "Mechanical Keyboard":  { img: "https://images.pexels.com/photos/1772123/pexels-photo-1772123.jpeg?auto=compress&cs=tinysrgb&w=400",   badge: "NEW",        bc: "blue",  cat: "Peripherals",  r: "4.9", rv: "182", old: "99.99" },
  "USB-C Hub":            { img: "https://images.pexels.com/photos/4195406/pexels-photo-4195406.jpeg?auto=compress&cs=tinysrgb&w=400",   badge: "BEST VALUE", bc: "green", cat: "Accessories",  r: "4.7", rv: "156", old: "64.99" },
};

document.addEventListener("DOMContentLoaded", () => {
  loadProducts();
  if (token) onLogin();
});

/* --- helpers --- */
function showToast(m) { const t = document.getElementById("toast"); t.textContent = m; t.classList.add("show"); setTimeout(() => t.classList.remove("show"), 3000); }
function showLoginModal() { document.getElementById("login-modal").style.display = "flex"; setTimeout(() => document.getElementById("email").focus(), 50); }
function hideLogin() { document.getElementById("login-modal").style.display = "none"; document.getElementById("login-err").textContent = ""; }
document.addEventListener("keydown", e => { if (e.key === "Escape") hideLogin(); });
function handleAcct() { token ? logout() : showLoginModal(); }

/* --- auth --- */
async function login(e) {
  e.preventDefault();
  const res = await fetch(API + "/login", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email: document.getElementById("email").value, password: document.getElementById("password").value }) });
  if (!res.ok) { document.getElementById("login-err").textContent = "Invalid credentials."; return; }
  token = (await res.json()).access_token;
  localStorage.setItem("token", token);
  hideLogin();
  onLogin();
  showToast("Welcome back!");
}

function onLogin() {
  document.getElementById("acct-text").textContent = "Sign Out";
  document.getElementById("orders-btn").style.display = "flex";
  document.getElementById("orders-section").style.display = "block";
  loadProducts();
  loadOrders();
}

function logout() {
  token = null; cartN = 0;
  localStorage.removeItem("token");
  document.getElementById("acct-text").textContent = "Sign In";
  document.getElementById("orders-btn").style.display = "none";
  document.getElementById("orders-section").style.display = "none";
  document.getElementById("cart-n").textContent = "0";
  loadProducts();
  showToast("Signed out.");
}

/* --- products --- */
async function loadProducts() {
  const res = await fetch(API + "/products");
  if (!res.ok) return;
  const products = await res.json();
  const c = document.getElementById("products");
  c.innerHTML = "";
  if (!products || !products.length) { c.innerHTML = '<p style="grid-column:1/-1;text-align:center;color:#aaa;padding:40px">No products.</p>'; return; }

  products.forEach(p => {
    const m = META[p.name] || { img: "", badge: "", bc: "blue", cat: "General", r: "4.5", rv: "0", old: "" };
    const d = document.createElement("div");
    d.className = "pc";
    d.innerHTML = `
      <div class="pc-top">
        ${m.badge ? '<span class="pc-badge ' + m.bc + '">' + m.badge + '</span>' : ''}
        <span class="pc-wish">&hearts;</span>
        ${m.img ? '<img class="pc-img" src="' + m.img + '" alt="' + esc(p.name) + '">' : ''}
      </div>
      <div class="pc-body">
        <div class="pc-cat">${esc(m.cat)}</div>
        <div class="pc-name">${esc(p.name)}</div>
        <div class="pc-desc">${esc(p.description?.String || '')}</div>
        <div class="pc-stars">${'\u2605'.repeat(Math.floor(m.r))}${m.r % 1 >= .5 ? '\u00BD' : ''}${'\u2606'.repeat(5 - Math.ceil(m.r))} <em>(${m.rv})</em></div>
        <div class="pc-prices">
          <span class="pc-price">$${p.price}</span>
          ${m.old ? '<span class="pc-old">$' + m.old + '</span>' : ''}
        </div>
        ${token
          ? '<button class="pc-add" onclick="order(this,\'' + esc(p.name) + '\',' + p.price + ')">Add to Cart</button>'
          : '<button class="pc-add" onclick="showLoginModal()">Sign in to Buy</button>'}
      </div>`;
    c.appendChild(d);
  });
}

/* --- orders --- */
async function order(btn, name, total) {
  btn.disabled = true; btn.textContent = "Processing...";
  const res = await fetch(API + "/orders", { method: "POST", headers: { "Content-Type": "application/json", Authorization: "Bearer " + token }, body: JSON.stringify({ total: parseFloat(total) }) });
  if (res.status === 401) { showToast("Session expired."); logout(); return; }
  if (!res.ok) { btn.disabled = false; btn.textContent = "Add to Cart"; showToast("Failed."); return; }
  cartN++; document.getElementById("cart-n").textContent = cartN;
  btn.textContent = "Added!"; btn.classList.add("done");
  showToast("Ordered " + name + " — $" + parseFloat(total).toFixed(2));
  setTimeout(() => { btn.disabled = false; btn.textContent = "Add to Cart"; btn.classList.remove("done"); }, 2000);
  loadOrders();
}

async function loadOrders() {
  const res = await fetch(API + "/orders", { headers: { Authorization: "Bearer " + token } });
  if (res.status === 401) { logout(); return; }
  if (!res.ok) return;
  const orders = await res.json();
  const c = document.getElementById("orders");
  if (!orders || !orders.length) { c.innerHTML = '<div class="oempty">No orders yet. Start shopping!</div>'; return; }
  cartN = orders.length;
  document.getElementById("cart-n").textContent = cartN;
  let h = '<div class="otbl"><div class="ohd"><span>Order</span><span>Date</span><span>Total</span><span>Status</span></div>';
  orders.forEach(o => {
    const dt = o.created_at?.Time ? new Date(o.created_at.Time).toLocaleDateString("en-US", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" }) : "";
    const st = o.status?.String || "pending";
    h += '<div class="orw"><span class="on">#' + o.id + '</span><span class="od">' + dt + '</span><span class="ot">$' + (o.total?.String || "0.00") + '</span><span class="bdg bdg-' + st + '">' + st + '</span></div>';
  });
  c.innerHTML = h + '</div>';
}

function esc(s) { const d = document.createElement("div"); d.textContent = s; return d.innerHTML; }
