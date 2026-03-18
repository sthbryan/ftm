<script>
  let { show, name, onConfirm, onCancel } = $props();
</script>

{#if show}
  <div
    class="modal-overlay"
    onclick={onCancel}
    role="presentation"
    tabindex="0"
    onkeydown={(e) => {
      if (
        e.key === "Enter" ||
        e.key === " " ||
        e.key === "Spacebar" ||
        e.key === "Escape"
      )
        onCancel();
    }}
  >
    <div
      class="modal"
      onclick={(e) => e.stopPropagation()}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
      tabindex="-1"
    >
      <div class="modal-header">
        <h3 id="modal-title">Delete Connection</h3>
        <button class="close-btn" onclick={onCancel} aria-label="Close dialog"
          >&times;</button
        >
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
    background: rgba(0, 0, 0, 0);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
    animation: overlayIn 0.25s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }

  @keyframes overlayIn {
    to {
      background: rgba(0, 0, 0, 0.5);
    }
  }

  .modal {
    background: white;
    border-radius: 16px;
    width: 90%;
    max-width: 400px;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
    transform: scale(0.9) translateY(20px);
    opacity: 0;
    animation: modalIn 0.35s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
  }

  @keyframes modalIn {
    to {
      transform: scale(1) translateY(0);
      opacity: 1;
    }
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 24px 24px 0;
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
    border-radius: 8px;
    transition: all 0.15s cubic-bezier(0.25, 1, 0.5, 1);
  }

  .close-btn:hover {
    background: #f5f5f4;
    transform: rotate(90deg);
  }

  .modal-body {
    padding: 16px 24px;
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
    padding: 0 24px 24px;
    justify-content: flex-end;
  }

  .btn {
    padding: 10px 20px;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    border: 1px solid #d6d3d1;
    background: white;
    color: #44403c;
    transition: all 0.15s cubic-bezier(0.25, 1, 0.5, 1);
    position: relative;
    overflow: hidden;
  }

  .btn:hover {
    background: #f5f5f4;
    transform: translateY(-1px);
  }

  .btn:active {
    transform: translateY(0) scale(0.98);
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
    box-shadow: 0 4px 12px rgba(220, 38, 38, 0.3);
  }

  .btn-danger:focus-visible,
  .btn-secondary:focus-visible {
    outline: 2px solid #92400e;
    outline-offset: 2px;
  }

  @media (max-width: 480px) {
    .modal {
      width: 95%;
      border-radius: 12px;
    }

    .modal-header {
      padding: 20px 20px 0;
    }

    .modal-body {
      padding: 14px 20px;
    }

    .modal-footer {
      padding: 0 20px 20px;
      flex-direction: column-reverse;
      gap: 8px;
    }

    .btn {
      width: 100%;
      padding: 12px;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .modal-overlay,
    .modal,
    .close-btn,
    .btn {
      animation: none;
      transform: none;
      transition: none;
    }
    .modal-overlay {
      background: rgba(0, 0, 0, 0.5);
    }
    .modal {
      opacity: 1;
    }
  }
</style>
