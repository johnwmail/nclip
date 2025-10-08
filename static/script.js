// Upload functionality
document.addEventListener('DOMContentLoaded', function () {
    const textContent = document.getElementById('text-content');
    const uploadTextBtn = document.getElementById('upload-text');
    const burnTextCheckbox = document.getElementById('burn-text');

    const fileInput = document.getElementById('file-input');
    const uploadFileBtn = document.getElementById('upload-file');
    const burnFileCheckbox = document.getElementById('burn-file');

    const resultSection = document.getElementById('result-section');
    const pasteUrlInput = document.getElementById('paste-url');
    const copyUrlBtn = document.getElementById('copy-url');
    const viewPasteLink = document.getElementById('view-paste');
    const rawPasteLink = document.getElementById('raw-paste');
    const newPasteBtn = document.getElementById('new-paste');

    // Capture initial usage examples HTML to restore exactly later
    const usageSection = document.querySelector('.usage-section');
    const usageExamplesContainer = document.querySelector('.code-examples');
    const initialUsageHTML = usageExamplesContainer ? usageExamplesContainer.innerHTML : '';

    // API key inputs (optional)
    const apiKeyInput = document.getElementById('api-key-input');
    const apiKeyInputMobile = document.getElementById('api-key-input-mobile');

    // Helper to get the API key value. Prefer mobile input if visible.
    function getApiKey() {
        try {
            if (apiKeyInputMobile) {
                // Check if mobile input is visible
                const style = window.getComputedStyle(apiKeyInputMobile);
                if (style && style.display !== 'none') {
                    return apiKeyInputMobile.value.trim();
                }
            }
        } catch (e) {
            // ignore
        }
        return apiKeyInput ? apiKeyInput.value.trim() : '';
    }

    // Text upload
    uploadTextBtn.addEventListener('click', function () {
        const content = textContent.value.trim();
        if (!content) {
            alert('Please enter some text to upload.');
            return;
        }

        const isBurn = burnTextCheckbox.checked;
        const endpoint = isBurn ? '/burn/' : '/';

        uploadTextBtn.disabled = true;
        uploadTextBtn.textContent = 'Uploading...';

        const headers = {
            'Content-Type': 'text/plain',
            'Accept': 'application/json',
        };
        const key = getApiKey();
        if (key) headers['Authorization'] = 'Bearer ' + key;

        fetch(endpoint, {
            method: 'POST',
            headers: headers,
            body: content
        })
            .then(async response => {
                if (!response.ok) {
                    const msg = await extractErrorMessage(response);
                    throw new Error(msg);
                }
                const data = await response.json();
                if (data.error) throw new Error(data.error);
                showResult(data.url, data.slug);
            })
            .catch(error => {
                alert('Upload failed: ' + error.message);
            })
            .finally(() => {
                uploadTextBtn.disabled = false;
                uploadTextBtn.textContent = 'Paste Text';
            });
    });

    // Helper: extract a safe error message from a Response object
    // Tries JSON first, then falls back to text, and always returns a string.
    async function extractErrorMessage(response) {
        const ct = response.headers.get('content-type') || '';
        try {
            // If content-type claims JSON, attempt parse
            if (ct.includes('application/json')) {
                const txt = await response.text();
                if (!txt) return `HTTP ${response.status} ${response.statusText}`;
                try {
                    const parsed = JSON.parse(txt);
                    if (parsed && typeof parsed === 'object') {
                        if (typeof parsed.error === 'string') return parsed.error;
                        if (typeof parsed.message === 'string') return parsed.message;
                        if (Array.isArray(parsed.errors) && parsed.errors.length) {
                            const first = parsed.errors[0];
                            if (typeof first === 'string') return first;
                            if (first && first.message) return first.message;
                        }
                    }
                    // fallback to raw text if structure not matched
                    return txt;
                } catch (e) {
                    // not valid JSON
                    return txt || `HTTP ${response.status} ${response.statusText}`;
                }
            }

            // Not JSON or no content-type: try text
            const text = await response.text();
            if (text && text.length) return text;
            return `HTTP ${response.status} ${response.statusText}`;
        } catch (e) {
            return `HTTP ${response.status} ${response.statusText}`;
        }
    }

    // File upload
    uploadFileBtn.addEventListener('click', function () {
        const file = fileInput.files[0];
        if (!file) {
            alert('Please select a file to upload.');
            return;
        }

        const isBurn = burnFileCheckbox.checked;
        const endpoint = isBurn ? '/burn/' : '/';
        const formData = new FormData();
        formData.append('file', file);

        uploadFileBtn.disabled = true;
        uploadFileBtn.textContent = 'Uploading...';

        const headers = { 'Accept': 'application/json' };
        const key = getApiKey();
        if (key) headers['Authorization'] = 'Bearer ' + key;

        fetch(endpoint, {
            method: 'POST',
            body: formData,
            headers: headers
        })
            .then(async response => {
                if (!response.ok) {
                    const msg = await extractErrorMessage(response);
                    throw new Error(msg);
                }
                const data = await response.json();
                if (data.error) throw new Error(data.error);
                showResult(data.url, data.slug);
            })
            .catch(error => {
                alert('Upload failed: ' + error.message);
            })
            .finally(() => {
                uploadFileBtn.disabled = false;
                uploadFileBtn.textContent = 'Upload File';
            });
    });

    // Show result
    function showResult(url, slug) {
        pasteUrlInput.value = url;
        if (viewPasteLink) {
            viewPasteLink.href = '/' + slug;
        }
        rawPasteLink.href = '/raw/' + slug;

        // Hide the upload section
        const uploadSection = document.querySelector('.upload-section');
        if (uploadSection) {
            uploadSection.style.display = 'none';
        }

        // Update usage examples to show how to read this paste
        updateUsageExamples(url, slug);

        resultSection.style.display = 'block';
        // Removed auto-scroll - let user stay at current position

        // Clear forms
        textContent.value = '';
        fileInput.value = '';
        burnTextCheckbox.checked = false;
        burnFileCheckbox.checked = false;
    }

    // Copy URL functionality
    copyUrlBtn.addEventListener('click', function () {
        pasteUrlInput.select();
        navigator.clipboard.writeText(pasteUrlInput.value).then(function () {
            const originalText = copyUrlBtn.textContent;
            copyUrlBtn.textContent = 'Copied!';
            copyUrlBtn.classList.add('success');
            setTimeout(function () {
                copyUrlBtn.textContent = originalText;
                copyUrlBtn.classList.remove('success');
            }, 2000);
        }).catch(function () {
            // Fallback for older browsers
            pasteUrlInput.select();
            document.execCommand('copy');
            const originalText = copyUrlBtn.textContent;
            copyUrlBtn.textContent = 'Copied!';
            copyUrlBtn.classList.add('success');
            setTimeout(function () {
                copyUrlBtn.textContent = originalText;
                copyUrlBtn.classList.remove('success');
            }, 2000);
        });
    });

    // New Paste button functionality
    newPasteBtn.addEventListener('click', function (event) {
        // Always reload the main page to ensure a pristine, server-rendered state
        event.preventDefault();
        window.location.assign('/');
    });


    // Update usage examples to show how to read the created paste
    function updateUsageExamples(url, slug) {
        const baseUrl = url.replace('/' + slug, '');
        const examples = document.querySelectorAll('.example');

        if (examples.length >= 2) {
            // Update first example - View paste
            examples[0].querySelector('p').textContent = 'View this paste:';
            examples[0].querySelector('code').textContent = `curl -sL ${url}`;

            // Update second example - Download raw
            examples[1].querySelector('p').textContent = 'Download raw content:';
            examples[1].querySelector('code').textContent = `curl -sL ${baseUrl}/raw/${slug}`;

            // Hide the third example if it exists
            if (examples[2]) {
                examples[2].style.display = 'none';
            }
        }
    }

    // Keyboard shortcuts
    document.addEventListener('keydown', function (e) {
        // Ctrl+Enter to upload text
        if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
            if (textContent.value.trim()) {
                uploadTextBtn.click();
            }
        }
    });

    // Drag and drop file upload
    document.addEventListener('dragover', function (e) {
        e.preventDefault();
        e.stopPropagation();
        document.body.classList.add('drag-over');
    });

    document.addEventListener('dragleave', function (e) {
        e.preventDefault();
        e.stopPropagation();
        if (e.clientX === 0 && e.clientY === 0) {
            document.body.classList.remove('drag-over');
        }
    });

    document.addEventListener('drop', function (e) {
        e.preventDefault();
        e.stopPropagation();
        document.body.classList.remove('drag-over');

        const files = e.dataTransfer.files;
        if (files.length > 0) {
            fileInput.files = files;
            // Optionally auto-upload
            // uploadFileBtn.click();
        }
    });
});

// Add some CSS for drag and drop
const style = document.createElement('style');
style.textContent = `
    .drag-over {
        background-color: #e3f2fd !important;
    }
    .drag-over::after {
        content: 'Drop file to upload';
        position: fixed;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        padding: 20px 40px;
        background-color: #2196f3;
        color: white;
        border-radius: 8px;
        font-weight: bold;
        z-index: 1000;
        pointer-events: none;
    }
`;
document.head.appendChild(style);