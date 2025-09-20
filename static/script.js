// Upload functionality
document.addEventListener('DOMContentLoaded', function() {
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

    // Text upload
    uploadTextBtn.addEventListener('click', function() {
        const content = textContent.value.trim();
        if (!content) {
            alert('Please enter some text to upload.');
            return;
        }

        const isBurn = burnTextCheckbox.checked;
        const endpoint = isBurn ? '/burn/' : '/';
        
        uploadTextBtn.disabled = true;
        uploadTextBtn.textContent = 'Uploading...';

        fetch(endpoint, {
            method: 'POST',
            headers: {
                'Content-Type': 'text/plain',
            },
            body: content
        })
        .then(response => response.json())
        .then(data => {
            if (data.error) {
                throw new Error(data.error);
            }
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

    // File upload
    uploadFileBtn.addEventListener('click', function() {
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

        fetch(endpoint, {
            method: 'POST',
            body: formData
        })
        .then(response => response.json())
        .then(data => {
            if (data.error) {
                throw new Error(data.error);
            }
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
    copyUrlBtn.addEventListener('click', function() {
        pasteUrlInput.select();
        navigator.clipboard.writeText(pasteUrlInput.value).then(function() {
            const originalText = copyUrlBtn.textContent;
            copyUrlBtn.textContent = 'Copied!';
            copyUrlBtn.classList.add('success');
            setTimeout(function() {
                copyUrlBtn.textContent = originalText;
                copyUrlBtn.classList.remove('success');
            }, 2000);
        }).catch(function() {
            // Fallback for older browsers
            pasteUrlInput.select();
            document.execCommand('copy');
            const originalText = copyUrlBtn.textContent;
            copyUrlBtn.textContent = 'Copied!';
            copyUrlBtn.classList.add('success');
            setTimeout(function() {
                copyUrlBtn.textContent = originalText;
                copyUrlBtn.classList.remove('success');
            }, 2000);
        });
    });

    // New Paste button functionality
    newPasteBtn.addEventListener('click', function(event) {
        // Always reload the main page to ensure a pristine, server-rendered state
        event.preventDefault();
        window.location.assign('/');
        return;
        
        // Legacy fallback (kept for reference, unreachable due to return above)
        // Hide result section
        resultSection.style.display = 'none';

        // Show upload section again
        const uploadSection = document.querySelector('.upload-section');
        if (uploadSection) {
            uploadSection.style.display = 'block';
        }

        // Reset all form fields
        textContent.value = '';
        fileInput.value = '';
        burnTextCheckbox.checked = false;
        burnFileCheckbox.checked = false;

        // Remove any file selection
        if (fileInput) {
            fileInput.value = '';
        }

        // Restore original usage examples exactly as initial render
        if (usageExamplesContainer) {
            usageExamplesContainer.innerHTML = initialUsageHTML;
        }

        // Show all usage examples (in case any were hidden)
        const examples = document.querySelectorAll('.example');
        examples.forEach(e => e.style.display = 'block');

        // Show the usage section if it was hidden
        if (usageSection) usageSection.style.display = 'block';

        // Scroll to top for a clean experience
        window.scrollTo({ top: 0, behavior: 'smooth' });

        // Focus on the text area
        textContent.focus();
    });

    // Update usage examples to show how to read the created paste
    function updateUsageExamples(url, slug) {
        const baseUrl = url.replace('/' + slug, '');
        const examples = document.querySelectorAll('.example');
        
        if (examples.length >= 2) {
            // Update first example - View paste
            examples[0].querySelector('p').textContent = 'View this paste:';
            examples[0].querySelector('code').textContent = `curl ${url}`;
            
            // Update second example - Download raw
            examples[1].querySelector('p').textContent = 'Download raw content:';
            examples[1].querySelector('code').textContent = `curl ${baseUrl}/raw/${slug}`;
            
            // Hide the third example if it exists
            if (examples[2]) {
                examples[2].style.display = 'none';
            }
        }
    }

    // Restore original usage examples for new paste
    function restoreOriginalUsageExamples() { /* no-op: handled by initialUsageHTML restore */ }

    // Keyboard shortcuts
    document.addEventListener('keydown', function(e) {
        // Ctrl+Enter to upload text
        if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
            if (textContent.value.trim()) {
                uploadTextBtn.click();
            }
        }
    });

    // Drag and drop file upload
    document.addEventListener('dragover', function(e) {
        e.preventDefault();
        e.stopPropagation();
        document.body.classList.add('drag-over');
    });

    document.addEventListener('dragleave', function(e) {
        e.preventDefault();
        e.stopPropagation();
        if (e.clientX === 0 && e.clientY === 0) {
            document.body.classList.remove('drag-over');
        }
    });

    document.addEventListener('drop', function(e) {
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