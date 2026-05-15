/**
 * Artie Main Application Logic
 * Handles folder selection, validation, and form population
 */

// Global state
let currentFolderData = null;
let currentFolderInfo = null;
let validationResult = null;

// Upload control
let uploadController = null; // For canceling uploads
let uploadCompleted = false; // Track if upload was completed successfully
let isUploading = false;

/**
 * Return the currently selected text format extension (sfm or usx)
 */
function getTextFormat() {
    const radio = document.querySelector('input[name="text_format"]:checked');
    return radio ? radio.value : 'sfm';
}

/**
 * Initialize folder upload functionality
 */
function initializeFolderUpload() {
    setupFolderDropzone();
    setupUploadButton();
}

/**
 * Setup folder dropzone (drag-and-drop + click-to-select)
 */
function setupFolderDropzone() {
    const dropzone = document.getElementById('folderDropzone');
    if (!dropzone) return;
    
    // Drag and drop events - must prevent default on ALL drag events
    dropzone.addEventListener('dragenter', function(e) {
        e.preventDefault();
        e.stopPropagation();
    });
    
    dropzone.addEventListener('dragover', function(e) {
        e.preventDefault();
        e.stopPropagation();
        e.dataTransfer.dropEffect = 'copy'; // Show copy cursor
        dropzone.classList.add('dragover');
    });
    
    dropzone.addEventListener('dragleave', function(e) {
        e.preventDefault();
        e.stopPropagation();
        dropzone.classList.remove('dragover');
    });
    
    dropzone.addEventListener('drop', function(e) {
        e.preventDefault();
        e.stopPropagation();
        dropzone.classList.remove('dragover');
        
        const items = e.dataTransfer.items;
        if (items && items.length > 0) {
            const item = items[0];
            if (item.kind === 'file') {
                // Try webkit API first (Chrome, Edge, Safari)
                if (item.webkitGetAsEntry) {
                    const entry = item.webkitGetAsEntry();
                    if (entry && entry.isDirectory) {
                        // It's a folder
                        showFolderStatus('📁 Processing dropped folder...', 'processing');
                        handleFolderSelection(entry);
                        return;
                    } else if (entry && entry.isFile) {
                        // It's a file - check if it's a YAML file
                        entry.file(function(file) {
                            if (file.name.toLowerCase().endsWith('.yaml') || file.name.toLowerCase().endsWith('.yml')) {
                                // Process as YAML file using existing logic
                                showFolderStatus('📄 Loading YAML file...', 'processing');
                                loadYAMLFile(file);
                            } else {
                                showFolderStatus('Please drop a folder or YAML file', 'error');
                            }
                        });
                        return;
                    }
                }
                
                // Try File System Access API (if available)
                if (item.getAsFileSystemHandle) {
                    item.getAsFileSystemHandle().then(handle => {
                        if (handle.kind === 'directory') {
                            showFolderStatus('📁 Processing dropped folder...', 'processing');
                            handleDirectoryPicker(handle);
                            return;
                        } else if (handle.kind === 'file') {
                            // It's a file - check if it's a YAML file
                            if (handle.name.toLowerCase().endsWith('.yaml') || handle.name.toLowerCase().endsWith('.yml')) {
                                showFolderStatus('📄 Loading YAML file...', 'processing');
                                handle.getFile().then(file => loadYAMLFile(file));
                            } else {
                                showFolderStatus('Please drop a folder or YAML file', 'error');
                            }
                            return;
                        }
                    }).catch(err => {
                        console.error('File System Access API error:', err);
                        showFolderStatus('Folder access not supported in this browser', 'error');
                    });
                    return;
                }
                
                // Fallback: try to get the file directly
                const file = item.getAsFile();
                if (file && (file.name.toLowerCase().endsWith('.yaml') || file.name.toLowerCase().endsWith('.yml'))) {
                    showFolderStatus('📄 Loading YAML file...', 'processing');
                    loadYAMLFile(file);
                    return;
                }
                
                // Fallback message
                showFolderStatus('Folder drag-and-drop not supported. Please use "Click to select"', 'error');
            } else {
                showFolderStatus('Please drop a folder or YAML file', 'error');
            }
        } else {
            showFolderStatus('No folder or file detected in drop', 'error');
        }
    });
    
    // Click to select folder
    dropzone.addEventListener('click', function() {
        if (window.showDirectoryPicker) {
            // Modern File System Access API
            showFolderStatus('📁 Please select a folder...', 'processing');
            window.showDirectoryPicker().then(handleDirectoryPicker)
                .catch(err => {
                    if (err.name !== 'AbortError') {
                        console.error('Directory picker error:', err);
                        showFolderStatus('Failed to select folder', 'error');
                    } else {
                        // User cancelled - reset to default state
                        showFolderStatus('No folder selected', 'default');
                    }
                });
        } else {
            // Fallback for older browsers
            showFolderStatus('📁 Please select a folder...', 'processing');
            const input = document.createElement('input');
            input.type = 'file';
            input.webkitdirectory = true;
            input.onchange = function(e) {
                if (e.target.files.length > 0) {
                    handleFileList(e.target.files);
                } else {
                    showFolderStatus('No folder selected', 'default');
                }
            };
            input.click();
        }
    });
}

/**
 * Load a YAML file using the same logic as the "Open" button
 */
function loadYAMLFile(file) {
    const reader = new FileReader();
    reader.onload = function(e) {
        try {
            const yamlContent = e.target.result;
            const jsYaml = window.jsyaml || window.yaml;
            
            if (!jsYaml) {
                showFolderStatus('YAML parser not loaded', 'error');
                return;
            }

            const data = jsYaml.load(yamlContent);
            
            // Use the existing populateForm function from arti.html
            if (typeof populateForm === 'function') {
                populateForm(data);
                
                // Update field styling to reflect loaded values
                if (typeof updateRequiredFieldStyling === 'function') {
                    updateRequiredFieldStyling();
                }
                
                // Update upload button state
                if (typeof updateUploadButtonState === 'function') {
                    updateUploadButtonState();
                }
                
                showFolderStatus(`📄 Loaded: ${file.name}`, 'success');
                
                // Clear folder-specific state since we loaded a YAML file
                currentFolderData = null;
                currentFolderInfo = null;
                validationResult = null;
                
                if (typeof clearUploadCelebration === 'function') {
                    clearUploadCelebration();
                }
            } else {
                showFolderStatus('Form population function not available', 'error');
            }
        } catch (error) {
            console.error('Error loading YAML file:', error);
            showFolderStatus('Error loading YAML: ' + error.message, 'error');
        }
    };
    reader.onerror = function() {
        showFolderStatus('Error reading file', 'error');
    };
    reader.readAsText(file);
}

/**
 * Handle directory picker selection (modern browsers)
 */
async function handleDirectoryPicker(directoryHandle) {
    try {
        const files = await collectFilesFSA(directoryHandle);
        const folderData = organizeFilesByType(files);
        const folderName = directoryHandle.name;
        processFolderData(folderData, folderName);
    } catch (error) {
        console.error('Error reading directory:', error);
        showFolderStatus('Failed to read folder contents', 'error');
    }
}

/**
 * Handle folder selection (called from drag-and-drop)
 */
async function handleFolderSelection(entry) {
    try {
        const folderData = await readDirectoryContentsWebkit(entry);
        const folderName = entry.name;
        processFolderData(folderData, folderName);
    } catch (error) {
        console.error('Error reading directory:', error);
        showFolderStatus('Failed to read folder contents', 'error');
    }
}

/**
 * Handle file list from webkitdirectory input (older browsers).
 * webkitRelativePath includes the root folder name (e.g. "N2MZJSIM/audio/a.mp3"),
 * so strip that prefix and store the remainder as customRelativePath so paths
 * are consistent with the drag-and-drop API path.
 */
function handleFileList(files) {
    const folderName = extractFolderNameFromPath(files[0]?.webkitRelativePath || '');
    const prefix = folderName ? `${folderName}/` : '';
    const normalised = Array.from(files).map(file => {
        const rel = file.webkitRelativePath || '';
        file.customRelativePath = rel.startsWith(prefix) ? rel.substring(prefix.length) : rel;
        return file;
    });
    const folderData = organizeFilesByType(normalised);
    processFolderData(folderData, folderName);
}

/**
 * Recursively collect all files from a directory handle, attaching
 * customRelativePath so organizeFilesByType can determine subfolder structure.
 */
async function collectFilesFSA(directoryHandle, currentPath = '') {
    const allFiles = [];

    for await (const [name, handle] of directoryHandle.entries()) {
        if (handle.kind === 'file') {
            const file = await handle.getFile();
            file.customRelativePath = currentPath ? `${currentPath}/${name}` : name;
            allFiles.push(file);
        } else if (handle.kind === 'directory') {
            const newPath = currentPath ? `${currentPath}/${name}` : name;
            const subFiles = await collectFilesFSA(handle, newPath);
            allFiles.push(...subFiles);
        }
    }

    return allFiles;
}

/**
 * Read directory contents using webkit API (for drag-and-drop)
 */
async function readDirectoryContentsWebkit(directoryEntry, currentPath = '') {
    const allFiles = [];
    
    const readDirectory = async (entry, path = '', depth = 0) => {
        // Prevent infinite recursion (safety limit)
        if (depth > 10) {
            console.warn(`⚠️  Maximum recursion depth reached at: ${path}/${entry.name}`);
            return;
        }
        
        try {
            if (entry.isFile) {
                const file = await new Promise((resolve, reject) => {
                    entry.file(resolve, reject);
                });
                // Create our own relative path since webkitRelativePath is unreliable
                file.customRelativePath = path ? `${path}/${entry.name}` : entry.name;
                
                // No debug logging for file discovery
                allFiles.push(file);
            } else if (entry.isDirectory) {
                const newPath = path ? `${path}/${entry.name}` : entry.name;
                
                const reader = entry.createReader();
                
                // Read all entries from this directory with robust error handling
                const readAllEntries = async () => {
                    try {
                        const entries = await new Promise((resolve, reject) => {
                            reader.readEntries(resolve, reject);
                        });
                        
                        if (entries.length === 0) {
                            return; // No more entries
                        }
                        
                        // Process each entry with error handling
                        for (const subEntry of entries) {
                            try {
                                await readDirectory(subEntry, newPath, depth + 1); // Recursive call with updated path and depth
                            } catch (entryError) {
                                console.error(`❌ Error processing entry ${subEntry.name}:`, entryError);
                                // Continue processing other entries even if one fails
                            }
                        }
                        
                        // Continue reading more entries (readEntries may not return all at once)
                        await readAllEntries();
                    } catch (readError) {
                        console.error(`❌ Error reading directory ${entry.name}:`, readError);
                        // Continue processing even if this directory fails
                    }
                };
                
                await readAllEntries();
            }
        } catch (error) {
            console.error(`❌ Error processing ${entry.name}:`, error);
            // Continue processing other entries even if this one fails
        }
    };
    
    // Iterate the root's children directly with an empty starting path so the
    // root folder name is not included in customRelativePath (folderName tracks it).
    const rootReader = directoryEntry.createReader();
    const readRootEntries = async () => {
        const entries = await new Promise((resolve, reject) => {
            rootReader.readEntries(resolve, reject);
        });
        if (entries.length === 0) return;
        for (const entry of entries) {
            await readDirectory(entry, '', 0);
        }
        await readRootEntries();
    };
    await readRootEntries();

    return organizeFilesByType(allFiles);
}

// Add new extensions here as needed — no other code needs to change.
const AUDIO_EXTS = new Set(['mp3', 'wav']);
const TEXT_EXTS  = new Set(['usx', 'sfm']);

/**
 * Return the lowercase extension of a filename, or null if unrecognised.
 */
function fileCategory(filename) {
    const ext = filename.split('.').pop().toLowerCase();
    if (AUDIO_EXTS.has(ext)) return 'audio';
    if (TEXT_EXTS.has(ext))  return 'text';
    return null;
}

/**
 * Organize files by type (audio, text).
 * Groups files by their actual parent directory path, validates that each
 * directory contains only one category (audio vs text), and returns the
 * discovered dirs with the actual extension case preserved from the files.
 * No assumptions are made about subfolder names or nesting depth.
 */
function organizeFilesByType(files) {
    // dirPath -> { category: 'audio'|'text'|null, files: [], extConflicts: Set }
    const dirMap = new Map();

    for (const file of files) {
        if (file.name.startsWith('.')) continue;

        const category = fileCategory(file.name);
        if (!category) continue; // skip unrecognised extensions

        const path = file.customRelativePath || '';
        const pathParts = path.split('/');
        const dirPath = pathParts.length >= 2 ? pathParts.slice(0, -1).join('/') : '';

        if (!dirMap.has(dirPath)) {
            dirMap.set(dirPath, { category: null, files: [], categories: new Set() });
        }

        const entry = dirMap.get(dirPath);
        entry.files.push(file);
        entry.categories.add(category);
    }

    const audioDirs  = [];
    const textDirs   = [];
    const errors     = [];
    const audioFiles = [];
    const textFiles  = [];

    for (const [dirPath, entry] of dirMap.entries()) {
        if (entry.categories.size === 0) continue;

        if (entry.categories.size > 1) {
            errors.push(`Directory "${dirPath || '(root)'}" contains mixed file types: ${[...entry.categories].join(', ')}`);
            continue;
        }

        const category = [...entry.categories][0];
        // Preserve the actual extension case from the first file in the directory
        const actualExt = entry.files[0].name.split('.').pop();

        if (category === 'audio') {
            audioDirs.push({ path: dirPath, ext: actualExt, files: entry.files });
            audioFiles.push(...entry.files);
        } else {
            textDirs.push({ path: dirPath, ext: actualExt, files: entry.files });
            textFiles.push(...entry.files);
        }
    }

    return {
        audioFiles,
        textFiles,
        audioDirs,
        textDirs,
        errors,
        totalFiles: audioFiles.length + textFiles.length
    };
}

/**
 * Extract folder name from file path
 */
function extractFolderNameFromPath(relativePath) {
    const parts = relativePath.split('/');
    return parts[0] || 'unknown';
}

/**
 * Process folder data (validate and populate form)
 */
async function processFolderData(folderData, folderName) {
    // Add folder name to data
    folderData.folderName = folderName;
    
    // Show initial feedback
    showFolderStatus('📁 Analyzing folder contents...', 'processing');
    updateFolderProgress(10, 100, 'Reading folder structure');
    
    // Parse folder name
    updateFolderProgress(30, 100, 'Parsing folder name');
    const folderInfo = parseFolderName(folderName);
    if (!folderInfo.valid) {
        showFolderStatus(folderInfo.error, 'error');
        return;
    }
    
    // Check for mixed-type directory errors before deeper validation
    if (folderData.errors && folderData.errors.length > 0) {
        showFolderStatus(`❌ Directory validation failed - click to see details`, 'error');
        showErrorModal(folderData.errors);
        currentFolderData = null;
        currentFolderInfo = null;
        validationResult = null;
        updateUploadButtonState();
        return;
    }

    // Validate folder structure
    updateFolderProgress(60, 100, 'Validating file structure');

    // Add a small delay to show the progress
    await new Promise(resolve => setTimeout(resolve, 200));

    const validation = validateFolderStructure(folderData);
    
    updateFolderProgress(90, 100, 'Finalizing validation');
    await new Promise(resolve => setTimeout(resolve, 100));
    
    if (validation.valid) {
        // Store data for upload
        currentFolderData = folderData;
        currentFolderInfo = folderInfo;
        validationResult = validation;
        
        // Populate form fields
        populateFormFromFolder(folderInfo, folderData);
        
        // Show success status
        showFolderStatus(`✅ Valid folder: ${validation.audioBooks.length} books, ${validation.totalAudioFiles} audio files, ${validation.totalTextFiles} text files`, 'success');
        
        // Enable upload button if credentials are loaded
        updateUploadButtonState();
        
    } else {
        // Show validation errors in modal
        showFolderStatus(`❌ Validation failed - click to see details`, 'error');
        showErrorModal(validation.errors);
        
        // Clear any previous data
        currentFolderData = null;
        currentFolderInfo = null;
        validationResult = null;
        updateUploadButtonState();
    }
}

/**
 * Populate form fields from folder data.
 * Uses the discovered audioDirs/textDirs — paths reflect the actual folder
 * structure with no assumptions about subfolder names or depth.
 */
function populateFormFromFolder(folderInfo, folderData) {
    const audioBucketName = getAudioBucketName();

    document.getElementById('datasetName').value = folderInfo.datasetName;
    document.getElementById('languageIso').value = folderInfo.iso;

    const audioDir = folderData.audioDirs && folderData.audioDirs[0];
    const textDir  = folderData.textDirs  && folderData.textDirs[0];

    const audioDirPart = audioDir && audioDir.path ? `${audioDir.path}/` : '';
    const textDirPart  = textDir  && textDir.path  ? `${textDir.path}/`  : '';

    document.getElementById('audioData').value = audioDir
        ? `s3://${audioBucketName}/${folderData.folderName}/${audioDirPart}*.${audioDir.ext}`
        : '';
    document.getElementById('textData').value = textDir
        ? `s3://${audioBucketName}/${folderData.folderName}/${textDirPart}*.${textDir.ext}`
        : '';

    // Set the text_format radio to match the discovered file type
    if (textDir) {
        const radioId = textDir.ext === 'usx' ? 'text_format_usx' : 'text_format_sfm';
        document.getElementById(radioId).checked = true;
    }

    clearFieldError('datasetName');
    clearFieldError('languageIso');
    clearFieldError('textData');
    clearFieldError('audioData');

    window.clearUploadCelebration();

    if (typeof updateRequiredFieldStyling === 'function') {
        updateRequiredFieldStyling();
    }
}

/**
 * Setup upload button functionality
 */
function setupUploadButton() {
    const uploadButton = document.getElementById('uploadButton');
    if (!uploadButton) return;
    
    uploadButton.addEventListener('click', async function(event) {
        console.log('🚀 Upload button clicked!');
        
        // Handle shift-click: download YAML without uploading
        if (event.shiftKey) {
            console.log('⬇️ Shift-click detected: downloading YAML only');
            
            // Check if we have a valid folder or all required fields
            const hasValidFolder = currentFolderData && currentFolderInfo && validationResult;
            const requiredFields = ['datasetName', 'username', 'languageIso', 'textData', 'audioData'];
            const allFieldsPopulated = requiredFields.every(fieldId => {
                const field = document.getElementById(fieldId);
                return field && field.value.trim().length > 0;
            });
            
            if (!hasValidFolder && !allFieldsPopulated) {
                console.log('❌ Missing folder data or required fields');
                showStatus('❌ Please select and validate a folder OR fill all required fields', 'error');
                return;
            }
            
            // Generate and download YAML
            saveFile();
            showStatus('📥 YAML downloaded successfully', 'success');
            return;
        }
        
        // Handle cancel upload
        if (isUploading) {
            console.log('🛑 Canceling upload...');
            if (uploadController) {
                uploadController.abort();
            }
            showStatus('Upload cancelled', 'warning');
            return;
        }
        
        console.log('  - currentFolderData:', !!currentFolderData);
        console.log('  - currentFolderInfo:', !!currentFolderInfo);
        console.log('  - validationResult:', !!validationResult);
        console.log('  - AWS credentials:', !!AWS_CONFIG.accessKeyId && !!AWS_CONFIG.secretAccessKey);
        
        // Check if we have either folder data OR all required fields populated
        const hasValidFolder = currentFolderData && currentFolderInfo && validationResult;
        const requiredFields = ['datasetName', 'username', 'languageIso', 'textData', 'audioData'];
        const allFieldsPopulated = requiredFields.every(fieldId => {
            const field = document.getElementById(fieldId);
            return field && field.value.trim().length > 0;
        });
        
        if (!hasValidFolder && !allFieldsPopulated) {
            console.log('❌ Missing folder data or required fields');
            showStatus('❌ Please select and validate a folder OR fill all required fields', 'error');
            return;
        }
        
        if (!AWS_CONFIG.accessKeyId || !AWS_CONFIG.secretAccessKey) {
            console.log('❌ Missing AWS credentials');
            showStatus('❌ Please load AWS credentials first', 'error');
            return;
        }
        
        // Check username
        const username = document.getElementById('username').value.trim();
        console.log('🔍 Username check:', `"${username}"`, 'Length:', username.length);
        if (!username) {
            console.log('❌ Missing username - showing error');
            showStatus('❌ Please enter a username (required field)', 'error');
            // Highlight the username field
            const usernameField = document.getElementById('username');
            usernameField.classList.add('required-error');
            usernameField.classList.remove('required-valid');
            usernameField.focus();
            return;
        }
        
        console.log('✅ All checks passed, starting upload...');
        
        // Show appropriate upload message based on upload type
        if (hasValidFolder) {
            // Warn about same folder upload
            const folderName = currentFolderData.folderName;
            showStatus(`⚠️ Uploading folder: ${folderName} (files with different sizes will be overwritten, same name and size will be skipped)`, 'warning');
        } else {
            // YAML upload message
            showStatus(`📤 Uploading YAML configuration...`, 'warning');
        }
        
        // Set upload state
        isUploading = true;
        uploadController = new AbortController();
        
        // Update button for upload state
        uploadButton.disabled = false;
        uploadButton.textContent = 'Cancel Upload';
        uploadButton.style.backgroundColor = '#dc3545'; // Red for cancel
        
        try {
            if (hasValidFolder) {
                // Folder upload: upload files + YAML
                const audioBucketName = getAudioBucketName();
                const yamlBucketName = getYamlBucketName();
                
                // Show progress bar
                const progressBar = document.getElementById('uploadProgressBar');
                if (progressBar) {
                    progressBar.style.display = 'block';
                }
                
                // Perform upload
                const result = await performUpload(currentFolderData, currentFolderInfo, audioBucketName, yamlBucketName, 
                    (current, total, message) => {
                        updateUploadProgress(current, total, message);
                    },
                    uploadController.signal
                );
                
                // Show success message with upload statistics
                const message = result.uploadedCount > 0 && result.skippedCount > 0 
                    ? `✅ Upload complete! Processed ${result.totalProcessed} files (${result.uploadedCount} uploaded, ${result.skippedCount} skipped) and YAML saved to s3://${yamlBucketName}/input/${currentFolderInfo.datasetName}.yaml`
                    : result.skippedCount === result.totalProcessed
                        ? `✅ Upload complete! All ${result.totalProcessed} files already exist (skipped) and YAML saved to s3://${yamlBucketName}/input/${currentFolderInfo.datasetName}.yaml`
                        : `✅ Upload complete! Files uploaded to s3://${audioBucketName}/${currentFolderData.folderName}/ and YAML saved to s3://${yamlBucketName}/input/${currentFolderInfo.datasetName}.yaml`;
                
                showStatus(message, 'success');
                
                // Also save locally
                saveFile();
                
            } else {
                // YAML-only upload: use progress bar and celebration like folder uploads
                const yamlBucketName = getYamlBucketName();
                const datasetName = document.getElementById('datasetName').value;
                
                // Show progress bar
                const progressBar = document.getElementById('uploadProgressBar');
                if (progressBar) {
                    progressBar.style.display = 'block';
                }
                
                // Generate YAML
                updateUploadProgress(10, 100, 'Generating YAML...');
                const yamlData = generateYAML();
                if (!yamlData) throw new Error('Failed to generate YAML');
                
                // Upload to S3
                updateUploadProgress(50, 100, 'Uploading to S3...');
                const s3 = new AWS.S3();
                const filename = `${datasetName}.yaml`;
                const params = {
                    Bucket: yamlBucketName,
                    Key: `input/${filename}`,
                    Body: yamlData,
                    ContentType: 'text/yaml'
                };
                
                const result = await s3.upload(params).promise();
                console.log('YAML upload successful:', result);
                
                // Show completion with celebration
                updateUploadProgress(100, 100, 'Upload complete!');
                
                // Show success message
                showStatus(`✅ YAML uploaded successfully to s3://${yamlBucketName}/input/${filename}`, 'success');
                
                // Also save locally
                saveFile();
            }
            
        } catch (error) {
            console.error('Upload error:', error);
            showStatus(`Upload failed: ${error.message}`, 'error');
        } finally {
            // Reset upload state
            isUploading = false;
            uploadController = null;
            
            // Reset button
            uploadButton.disabled = false;
            uploadButton.textContent = 'Upload';
            uploadButton.style.backgroundColor = ''; // Reset to default color
            
            // Note: Progress bar is NOT hidden here to allow celebration to persist
            // It will be hidden when user makes changes or clicks Clear
        }
    });
}

/**
 * Update upload button state based on current conditions
 */
// Make function globally accessible
window.updateUploadButtonState = function() {
    const uploadButton = document.getElementById('uploadButton');
    if (!uploadButton) {
        console.log('❌ Upload button not found');
        return;
    }
    
    const hasCredentials = AWS_CONFIG.accessKeyId && AWS_CONFIG.secretAccessKey;
    const hasValidFolder = currentFolderData && currentFolderInfo && validationResult;
    
    // Check if all required fields are populated (for YAML files or manual entry)
    const requiredFields = ['datasetName', 'username', 'languageIso', 'textData', 'audioData'];
    const allFieldsPopulated = requiredFields.every(fieldId => {
        const field = document.getElementById(fieldId);
        return field && field.value.trim().length > 0;
    });
    
    console.log('🔍 Upload button state check:');
    console.log('  - Credentials loaded:', hasCredentials);
    console.log('  - Valid folder:', hasValidFolder);
    console.log('  - All required fields populated:', allFieldsPopulated);
    
    // Ready if: (has credentials) AND all required fields populated
    // Note: Whether fields are populated from folder or manually doesn't matter
    const isReady = hasCredentials && allFieldsPopulated;
    uploadButton.disabled = !isReady;
    
    // Remove existing styling classes
    uploadButton.classList.remove('upload-error', 'upload-ready');
    
    if (!hasCredentials) {
        uploadButton.title = 'Load AWS credentials first';
        uploadButton.classList.add('upload-error');
        console.log('  - Button disabled: Missing credentials');
    } else if (!allFieldsPopulated) {
        uploadButton.title = 'Fill all required fields (username, dataset name, language ISO, text data, audio data)';
        uploadButton.classList.add('upload-error');
        console.log('  - Button disabled: Missing required fields');
    } else {
        uploadButton.title = 'Upload to S3';
        uploadButton.classList.add('upload-ready');
        console.log('  - Button ENABLED: All requirements met');
    }
};

/**
 * Show folder status message
 */
function showFolderStatus(message, type) {
    const statusDiv = document.getElementById('folderStatus');
    const dropzone = document.getElementById('folderDropzone');
    
    if (statusDiv) {
        statusDiv.textContent = message;
        statusDiv.className = `folder-status ${type}`;
    }
    
    if (dropzone) {
        dropzone.className = `folder-dropzone ${type}`;
    }
    
    // Hide progress bar when not processing
    if (type !== 'processing') {
        hideFolderProgress();
    }
}

/**
 * Update folder validation progress
 */
function updateFolderProgress(current, total, message) {
    const progressDiv = document.getElementById('folderProgress');
    if (!progressDiv) return;
    
    const percentage = Math.round((current / total) * 100);
    progressDiv.style.display = 'block';
    progressDiv.innerHTML = `
        <div class="progress-bar">
            <div class="progress-fill" style="width: ${percentage}%"></div>
        </div>
        <div class="progress-text">${message} (${percentage}%)</div>
    `;
}

/**
 * Hide folder progress indicator
 */
function hideFolderProgress() {
    const progressDiv = document.getElementById('folderProgress');
    if (progressDiv) {
        progressDiv.style.display = 'none';
    }
}

/**
 * Update upload progress display
 */
function updateUploadProgress(current, total, message) {
    const progressBar = document.getElementById('uploadProgressBar');
    const progressFill = document.getElementById('uploadProgressFill');
    const progressText = document.getElementById('uploadProgressText');
    const progressMessage = document.getElementById('uploadProgressMessage');
    
    if (progressBar && progressFill && progressText && progressMessage) {
        const percentage = Math.round((current / total) * 100);
        
        // Show progress bar if hidden
        if (progressBar.style.display === 'none') {
            progressBar.style.display = 'block';
        }
        
        // Check if upload is complete
        if (percentage === 100) {
            // Replace progress bar with celebratory completion message
            progressFill.style.width = `100%`;
            progressFill.style.backgroundColor = '#00ff00'; // Bright green
            progressText.textContent = `🎉 100% 🎉`;
            progressMessage.innerHTML = `🎊 <strong style="color: #00ff00; font-size: 16px;">Upload Completed!</strong> 🎊`;
            uploadCompleted = true; // Mark upload as completed
        } else {
            // Normal progress update
            progressFill.style.width = `${percentage}%`;
            progressFill.style.backgroundColor = '#4CAF50'; // Normal green
            progressText.textContent = `${percentage}%`;
            
            // Show only the last uploaded file name (no file count)
            if (message) {
                progressMessage.textContent = message;
            } else {
                progressMessage.textContent = `Uploading...`;
            }
        }
    }
}

/**
 * Clear field error styling
 */
function clearFieldError(fieldId) {
    const element = document.getElementById(fieldId);
    if (element) {
        element.classList.remove('required-error');
    }
}

/**
 * Clear upload completion celebration
 */
window.clearUploadCelebration = function() {
    console.log('clearUploadCelebration called, uploadCompleted:', uploadCompleted);
    if (uploadCompleted) {
        uploadCompleted = false;
        const progressBar = document.getElementById('uploadProgressBar');
        if (progressBar) {
            console.log('Hiding progress bar');
            progressBar.style.display = 'none';
        } else {
            console.log('Progress bar element not found');
        }
    }
};

// Prevent default drag-and-drop behavior globally to avoid browser opening files
// Use capture phase (true) to catch events before they reach target elements
document.addEventListener('dragenter', function(e) {
    e.preventDefault();
}, true);

document.addEventListener('dragover', function(e) {
    e.preventDefault();
}, true);

document.addEventListener('drop', function(e) {
    // Don't prevent default here - let the dropzone handle it
}, true);

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    // Wait a bit for other scripts to load
    setTimeout(initializeFolderUpload, 100);
});
