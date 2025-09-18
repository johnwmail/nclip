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
            uploadTextBtn.textContent = 'Upload Text';
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
        viewPasteLink.href = '/' + slug;
        rawPasteLink.href = '/raw/' + slug;
        
        // Hide the upload section
        const uploadSection = document.querySelector('.upload-section');
        if (uploadSection) {
            uploadSection.style.display = 'none';
        }
        
        // Update usage examples to show how to read this paste
        updateUsageExamples(url, slug);
        
        resultSection.style.display = 'block';
        resultSection.scrollIntoView({ behavior: 'smooth' });
        
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
    newPasteBtn.addEventListener('click', function() {
        // Hide result section
        resultSection.style.display = 'none';
        
        // Show upload section again
        const uploadSection = document.querySelector('.upload-section');
        if (uploadSection) {
            uploadSection.style.display = 'block';
        }
        
        // Clear forms and focus on text area
        textContent.value = '';
        fileInput.value = '';
        burnTextCheckbox.checked = false;
        burnFileCheckbox.checked = false;
        textContent.focus();
        
        // Restore original usage examples
        restoreOriginalUsageExamples();
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
    function restoreOriginalUsageExamples() {
        const examples = document.querySelectorAll('.example');
        
        if (examples.length >= 3) {
            // Get the base URL from current location
            const baseUrl = window.location.origin;
            
            // Restore original examples
            examples[0].querySelector('p').textContent = 'Upload text:';
            examples[0].querySelector('code').textContent = `echo "hello world" | curl --data-binary @- ${baseUrl}`;
            
            examples[1].querySelector('p').textContent = 'Upload file:';
            examples[1].querySelector('code').textContent = `curl --data-binary @/path/to/file ${baseUrl}`;
            
            examples[2].querySelector('p').textContent = 'Burn after reading:';
            examples[2].querySelector('code').textContent = `echo "secret" | curl --data-binary @- ${baseUrl}/burn/`;
            
            // Show all examples
            examples[0].style.display = 'block';
            examples[1].style.display = 'block';
            examples[2].style.display = 'block';
        }
    }

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