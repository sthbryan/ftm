<script>
  let { show, name, onConfirm, onCancel } = $props();
</script>

{#if show}
  <div class="modal-overlay" onclick={onCancel}>
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h3>Delete Connection</h3>
        <button class="close-btn" onclick={onCancel}>&times;</button>
      </div>
      <div class="modal-body">
        <p>Are you sure you want to delete <strong>{name}</strong>?</p>
        <p class="warning">This action cannot be undone.</p>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" onclick={onCancel}>Cancel</button>
        <button class="btn btn-danger" onclick={onConfirm}>Delete</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
    animation: fadeIn 0.15s ease;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .modal {
    background: white;
    border-radius: 12px;
    width: 90%;
    max-width: 400px;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
    animation: slideIn 0.15s ease;
  }

  @keyframes slideIn {
    from { transform: translateY(-20px); opacity: 0; }
    to { transform: translateY(0); opacity: 1; }
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px 20px 0;
  }

  .modal-header h3 {
    font-size: 18px;
    font-weight: 600;
    color: #1c1917;
    margin: 0;
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 24px;
    color: #78716c;
    cursor: pointer;
    padding: 0;
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
  }

  .close-btn:hover {
    background: #f5f5f4;
  }

  .modal-body {
    padding: 16px 20px;
  }

  .modal-body p {
    margin: 0 0 8px;
    color: #44403c;
  }

  .warning {
    font-size: 13px;
    color: #78716c;
  }

  .modal-footer {
    display: flex;
    gap: 12px;
    padding: 0 20px 20px;
    justify-content: flex-end;
  }

  .btn {
    padding: 10px 16px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    border: 1px solid #d6d3d1;
    background: white;
    color: #44403c;
    transition: all 0.15s;
  }

  .btn:hover {
    background: #f5f5f4;
  }

  .btn-secondary {
    border-color: #d6d3d1;
  }

  .btn-danger {
    background: #dc2626;
    color: white;
    border-color: #dc2626;
  }

  .btn-danger:hover {
    background: #b91c1c;
    border-color: #b91c1c;
  }
</style>
