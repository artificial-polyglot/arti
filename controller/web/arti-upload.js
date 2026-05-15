/**
 * Artie Upload Functionality
 * Handles S3 upload with progress tracking and YAML generation
 */

/**
 * Return the relative directory path stored on a file object.
 * customRelativePath is set by collectFilesFSA / readDirectoryContentsWebkit
 * and is always relative to the root folder (root folder name excluded).
 */
function fileRelativeDir(file) {
    const relPath = file.customRelativePath || '';
    const parts = relPath.split('/');
    return parts.length >= 2 ? parts.slice(0, -1).join('/') : '';
}

/**
 * Build the S3 key for a file: folderName + the file's own relative path.
 * No assumptions are made about subfolder names or depth.
 */
function fileS3Key(folderName, file) {
    const relDir = fileRelativeDir(file);
    return relDir ? `${folderName}/${relDir}/${file.name}` : `${folderName}/${file.name}`;
}

/**
 * Check existing files in S3 bucket for smart re-upload.
 * Lists everything under folderName/ and keys the map by relative path
 * (without the leading folderName/) so it works for any directory structure.
 */
async function checkExistingFiles(folderData, folderName, audioBucketName) {
    const s3 = new AWS.S3();
    const existingFiles = new Map(); // relKey -> size

    try {
        const listResult = await s3.listObjectsV2({
            Bucket: audioBucketName,
            Prefix: `${folderName}/`
        }).promise();

        if (listResult.Contents) {
            listResult.Contents.forEach(obj => {
                // Strip "folderName/" prefix to get the relative key
                const relKey = obj.Key.substring(folderName.length + 1);
                existingFiles.set(relKey, obj.Size);
            });
        }

        return existingFiles;
    } catch (error) {
        console.warn('Could not check existing files, will upload all:', error.message);
        return new Map();
    }
}

/**
 * Upload files to S3 with progress tracking and smart re-upload.
 * Each file is uploaded to the path it was discovered at — no assumptions
 * about subfolder names or nesting depth.
 */
async function uploadFilesToS3(folderData, folderName, audioBucketName, onProgress, abortSignal = null) {
    const s3 = new AWS.S3();
    const uploadedFiles = [];
    const totalFiles = folderData.audioFiles.length + folderData.textFiles.length;
    let uploadedCount = 0;
    let skippedCount = 0;

    // Check existing files first (keyed by relative path, not just filename)
    onProgress(0, totalFiles, 'Checking existing files...');
    const existingFiles = await checkExistingFiles(folderData, folderName, audioBucketName);

    // Upload audio files
    for (const file of folderData.audioFiles) {
        if (abortSignal && abortSignal.aborted) throw new Error('Upload cancelled');

        const relDir = fileRelativeDir(file);
        const relKey = relDir ? `${relDir}/${file.name}` : file.name;
        const key = `${folderName}/${relKey}`;
        const existingSize = existingFiles.get(relKey);

        if (existingSize !== undefined && existingSize === file.size) {
            skippedCount++;
            uploadedCount++;
            uploadedFiles.push(key);
            if (onProgress) onProgress(uploadedCount, totalFiles, `Skipped: ${file.name} (already exists)`);
            continue;
        }

        try {
            const fileContent = await readFileAsArrayBuffer(file);
            await s3.upload({
                Bucket: audioBucketName,
                Key: key,
                Body: fileContent,
                ContentType: getContentType(file.name)
            }).promise();

            uploadedFiles.push(key);
            uploadedCount++;
            if (onProgress) onProgress(uploadedCount, totalFiles, `Uploaded: ${file.name}`);
        } catch (error) {
            if (abortSignal && abortSignal.aborted) throw new Error('Upload cancelled');
            throw new Error(`Failed to upload audio file ${file.name}: ${error.message}`);
        }
    }

    // Upload text files
    for (const file of folderData.textFiles) {
        if (abortSignal && abortSignal.aborted) throw new Error('Upload cancelled');

        const relDir = fileRelativeDir(file);
        const relKey = relDir ? `${relDir}/${file.name}` : file.name;
        const key = `${folderName}/${relKey}`;
        const existingSize = existingFiles.get(relKey);

        if (existingSize !== undefined && existingSize === file.size) {
            skippedCount++;
            uploadedCount++;
            uploadedFiles.push(key);
            if (onProgress) onProgress(uploadedCount, totalFiles, `Skipped: ${file.name} (already exists)`);
            continue;
        }

        try {
            const fileContent = await readFileAsArrayBuffer(file);
            await s3.upload({
                Bucket: audioBucketName,
                Key: key,
                Body: fileContent,
                ContentType: getContentType(file.name)
            }).promise();

            uploadedFiles.push(key);
            uploadedCount++;
            if (onProgress) onProgress(uploadedCount, totalFiles, `Uploaded: ${file.name}`);
        } catch (error) {
            if (abortSignal && abortSignal.aborted) throw new Error('Upload cancelled');
            throw new Error(`Failed to upload text file ${file.name}: ${error.message}`);
        }
    }

    return {
        uploadedFiles,
        uploadedCount: uploadedCount - skippedCount,
        skippedCount,
        totalProcessed: uploadedCount
    };
}

/**
 * Read file as ArrayBuffer (for binary files)
 */
function readFileAsArrayBuffer(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(reader.result);
        reader.onerror = () => reject(new Error('Failed to read file'));
        reader.readAsArrayBuffer(file);
    });
}

/**
 * Get content type based on file extension
 */
function getContentType(filename) {
    const ext = filename.toLowerCase().split('.').pop();
    switch (ext) {
        case 'mp3': return 'audio/mpeg';
        case 'wav': return 'audio/wav';
        case 'usx': return 'text/xml';
        case 'sfm': return 'text/plain';
        default: return 'application/octet-stream';
    }
}

/**
 * Generate YAML content for the request.
 * Uses the discovered audioDirs and textDirs from the folder scan — the paths
 * in the YAML exactly match where the files were uploaded.
 */
function generateUploadYAML(folderData, folderInfo, audioBucketName) {
    const yamlData = window.buildYAMLObject();
    if (!yamlData) return null;

    const audioDir = folderData.audioDirs && folderData.audioDirs[0];
    const textDir = folderData.textDirs && folderData.textDirs[0];

    if (audioDir) {
        const dirPart = audioDir.path ? `${audioDir.path}/` : '';
        yamlData.audio_data = { aws_s3: `s3://${audioBucketName}/${folderData.folderName}/${dirPart}*.${audioDir.ext}` };
    }
    if (textDir) {
        const dirPart = textDir.path ? `${textDir.path}/` : '';
        yamlData.text_data = { aws_s3: `s3://${audioBucketName}/${folderData.folderName}/${dirPart}*.${textDir.ext}` };
    }

    return yamlData;
}

/**
 * Upload YAML file to S3
 */
async function uploadYAMLToS3(yamlData, datasetName, yamlBucketName) {
    const s3 = new AWS.S3();

    let yamlString;
    if (window.jsyaml) {
        yamlString = window.jsyaml.dump(yamlData, { indent: 4, lineWidth: -1 });
    } else {
        yamlString = generateSimpleYAML(yamlData);
    }

    // Ensure csv: yes is unquoted
    yamlString = yamlString.replace(/^(\s+)csv:\s*['"]yes['"]/m, '$1csv: yes');

    const key = `input/${datasetName}.yaml`;

    await s3.upload({
        Bucket: yamlBucketName,
        Key: key,
        Body: yamlString,
        ContentType: 'text/yaml'
    }).promise();

    return key;
}

/**
 * Simple YAML generator (fallback when js-yaml is not available)
 */
function generateSimpleYAML(obj) {
    let yaml = '';

    function processValue(key, value, indent = 0) {
        const spaces = '    '.repeat(indent);

        if (value === null || value === undefined) return '';
        if (typeof value === 'boolean') return `${spaces}${key}: ${value}`;
        if (typeof value === 'number') return `${spaces}${key}: ${value}`;
        if (typeof value === 'string') return `${spaces}${key}: ${value}`;

        if (Array.isArray(value)) {
            let result = `${spaces}${key}:\n`;
            value.forEach(item => { result += `${spaces}    - ${item}\n`; });
            return result;
        }

        if (typeof value === 'object') {
            let result = `${spaces}${key}:\n`;
            for (const [k, v] of Object.entries(value)) {
                result += processValue(k, v, indent + 1) + '\n';
            }
            return result;
        }

        return `${spaces}${key}: ${value}`;
    }

    for (const [key, value] of Object.entries(obj)) {
        yaml += processValue(key, value) + '\n';
    }

    return yaml;
}

/**
 * Main upload function
 */
async function performUpload(folderData, folderInfo, audioBucketName, yamlBucketName, onProgress, abortSignal = null) {
    try {
        onProgress(0, 100, 'Starting file upload...');
        const uploadResult = await uploadFilesToS3(folderData, folderData.folderName, audioBucketName,
            (current, total, message) => {
                const percentage = Math.round((current / total) * 80);
                onProgress(percentage, 100, message);
            },
            abortSignal
        );

        onProgress(85, 100, 'Generating YAML...');
        const yamlData = generateUploadYAML(folderData, folderInfo, audioBucketName);

        onProgress(90, 100, 'Uploading YAML...');
        const currentDatasetName = document.getElementById('datasetName').value || folderInfo.datasetName;
        const yamlKey = await uploadYAMLToS3(yamlData, currentDatasetName, yamlBucketName);

        onProgress(100, 100, 'Upload complete!');

        return {
            success: true,
            uploadedFiles: uploadResult.uploadedFiles,
            yamlKey,
            uploadedCount: uploadResult.uploadedCount,
            skippedCount: uploadResult.skippedCount,
            totalProcessed: uploadResult.totalProcessed,
            message: `Successfully processed ${uploadResult.totalProcessed} files (${uploadResult.uploadedCount} uploaded, ${uploadResult.skippedCount} skipped) and YAML configuration`
        };

    } catch (error) {
        throw new Error(`Upload failed: ${error.message}`);
    }
}
