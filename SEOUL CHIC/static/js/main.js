/* ═══════════════════════════════════════════════════════════════════
   Seoul Chic — main.js
   Client-side: password toggle, form interactions, order confirm UI
═══════════════════════════════════════════════════════════════════ */

document.addEventListener('DOMContentLoaded', function () {

  // ── Password visibility toggle ──────────────────────────────────
  document.querySelectorAll('#togglePass').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var input = this.closest('.input-group').querySelector('input[type="password"], input[type="text"]');
      if (!input) return;
      var isPass = input.type === 'password';
      input.type = isPass ? 'text' : 'password';
      this.querySelector('i').className = isPass ? 'bi bi-eye-slash' : 'bi bi-eye';
    });
  });

  // ── Register: password match feedback ─────────────────────────
  var pwField      = document.getElementById('password');
  var confirmField = document.getElementById('confirm_password');
  var feedback     = document.getElementById('pwMatchFeedback');
  if (pwField && confirmField && feedback) {
    function checkMatch() {
      if (confirmField.value === '') { feedback.textContent = ''; return; }
      if (pwField.value === confirmField.value) {
        feedback.textContent = '✓ รหัสผ่านตรงกัน';
        feedback.className   = 'form-text text-success';
      } else {
        feedback.textContent = '✗ รหัสผ่านไม่ตรงกัน';
        feedback.className   = 'form-text text-danger';
      }
    }
    confirmField.addEventListener('input', checkMatch);
    pwField.addEventListener('input', checkMatch);
  }

  // ── Order Confirm: delivery type toggle ────────────────────────
  var orderTypeRadios  = document.querySelectorAll('.order-type-radio');
  var deliveryBlock    = document.getElementById('deliveryAddrBlock');
  var deliveryTextarea = document.getElementById('delivery_address');

  orderTypeRadios.forEach(function (radio) {
    radio.addEventListener('change', function () {
      var isDelivery = this.value === 'delivery';
      if (deliveryBlock) {
        deliveryBlock.classList.toggle('d-none', !isDelivery);
        if (deliveryTextarea) deliveryTextarea.required = isDelivery;
      }
      // Update card selection highlight
      document.querySelectorAll('.order-type-card').forEach(function (card) {
        card.classList.remove('selected');
      });
      this.closest('.order-type-card').classList.add('selected');
    });
    // Trigger on load for pre-selected
    if (radio.checked) radio.dispatchEvent(new Event('change'));
  });

  // ── Order Confirm: payment method card highlight ───────────────
  document.querySelectorAll('.payment-radio').forEach(function (radio) {
    radio.addEventListener('change', function () {
      document.querySelectorAll('.payment-card').forEach(function (c) {
        c.classList.remove('selected');
      });
      this.closest('.payment-card').classList.add('selected');
    });
    if (radio.checked) radio.dispatchEvent(new Event('change'));
  });

  // ── Clickable table rows ────────────────────────────────────────
  document.querySelectorAll('.clickable-row[data-href]').forEach(function (row) {
    row.addEventListener('click', function (e) {
      // Don't navigate if a button/link inside the row was clicked
      if (e.target.closest('a, button, form')) return;
      window.location.href = this.dataset.href;
    });
  });

  // ── Auto-dismiss alerts after 5 seconds ────────────────────────
  document.querySelectorAll('.kfry-alert').forEach(function (el) {
    setTimeout(function () {
      var bsAlert = bootstrap.Alert.getOrCreateInstance(el);
      if (bsAlert) bsAlert.close();
    }, 5000);
  });

  // ── Add-to-cart button feedback ────────────────────────────────
  document.querySelectorAll('.add-cart-form').forEach(function (form) {
    form.addEventListener('submit', function () {
      var btn = this.querySelector('button[type="submit"]');
      if (btn) {
        btn.innerHTML = '<i class="bi bi-check-lg"></i>';
        btn.disabled = true;
      }
    });
  });

  // ── Admin: confirm delete forms ────────────────────────────────
  // (handled inline with onsubmit, but also support data-confirm)
  document.querySelectorAll('[data-confirm]').forEach(function (el) {
    el.addEventListener('click', function (e) {
      if (!confirm(this.dataset.confirm)) e.preventDefault();
    });
  });

});
