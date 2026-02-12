/**
 * Point Cloud Annotator
 * 
 * A web application for annotating 3D point clouds using Potree.
 * Supports creating, viewing, and deleting annotations with persistence.
 */

import * as THREE from "/potree/libs/three.js/build/three.module.js";

// API Configuration
const API_BASE_URL = window.location.origin + '/api/v1';

// Application State
const state = {
    isAddingAnnotation: false,
    annotations: new Map(), // Map of annotation ID to Potree.Annotation
    pendingPosition: null,
};

// DOM Elements
const elements = {
    dialog: document.getElementById('annotation-dialog'),
    dialogTitle: document.getElementById('dialog-title'),
    titleInput: document.getElementById('annotation-title'),
    descriptionInput: document.getElementById('annotation-description'),
    saveButton: document.getElementById('dialog-save'),
    cancelButton: document.getElementById('dialog-cancel'),
    addButton: document.getElementById('btn-add-annotation'),
    refreshButton: document.getElementById('btn-refresh'),
    statusMessage: document.getElementById('status-message'),
    annotationCount: document.getElementById('annotation-count'),
};

/**
 * API Client for annotation operations
 */
const api = {
    async getAnnotations() {
        const response = await fetch(`${API_BASE_URL}/annotations`);
        if (!response.ok) {
            throw new Error(`Failed to fetch annotations: ${response.statusText}`);
        }
        const data = await response.json();
        return data.data || [];
    },

    async createAnnotation(annotation) {
        const response = await fetch(`${API_BASE_URL}/annotations`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(annotation),
        });
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.message || 'Failed to create annotation');
        }
        const data = await response.json();
        return data.data;
    },

    async updateAnnotation(id, updates) {
        const response = await fetch(`${API_BASE_URL}/annotations/${id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(updates),
        });
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.message || 'Failed to update annotation');
        }
        const data = await response.json();
        return data.data;
    },

    async deleteAnnotation(id) {
        const response = await fetch(`${API_BASE_URL}/annotations/${id}`, {
            method: 'DELETE',
        });
        if (!response.ok && response.status !== 204) {
            const error = await response.json();
            throw new Error(error.message || 'Failed to delete annotation');
        }
    },
};

/**
 * Update the status message
 */
function setStatus(message, type = 'info') {
    elements.statusMessage.textContent = message;
    elements.statusMessage.className = type;
    
    // Auto-clear success/error messages after 3 seconds
    if (type !== 'info') {
        setTimeout(() => {
            elements.statusMessage.textContent = 'Ready';
            elements.statusMessage.className = '';
        }, 3000);
    }
}

/**
 * Update the annotation count display
 */
function updateAnnotationCount() {
    elements.annotationCount.textContent = `Annotations: ${state.annotations.size}`;
}

/**
 * Show the annotation dialog
 */
function showDialog(title = 'Create Annotation', existingData = null) {
    elements.dialogTitle.textContent = title;
    elements.titleInput.value = existingData?.title || '';
    elements.descriptionInput.value = existingData?.description || '';
    elements.dialog.classList.remove('hidden');
    elements.titleInput.focus();
}

/**
 * Hide the annotation dialog
 */
function hideDialog() {
    elements.dialog.classList.add('hidden');
    elements.titleInput.value = '';
    elements.descriptionInput.value = '';
    state.pendingPosition = null;
}

/**
 * Create a Potree annotation from API data
 */
function createPotreeAnnotation(annotationData) {
    const annotation = new Potree.Annotation({
        position: [annotationData.x, annotationData.y, annotationData.z],
        title: annotationData.title,
        description: annotationData.description || '',
    });

    // Store the API ID on the annotation object
    annotation.apiId = annotationData.id;

    // Add custom class for styling
    annotation.domElement.addClass('annotation-custom');

    // Create delete button
    const deleteBtn = $(`<button class="annotation-delete-btn" title="Delete annotation">&times;</button>`);
    deleteBtn.on('click', async (e) => {
        e.stopPropagation();
        await deleteAnnotation(annotationData.id);
    });
    annotation.domElement.find('.annotation-titlebar').append(deleteBtn);

    // Store reference
    state.annotations.set(annotationData.id, annotation);

    return annotation;
}

/**
 * Load all annotations from the API
 */
async function loadAnnotations() {
    setStatus('Loading annotations...');
    
    try {
        const annotations = await api.getAnnotations();
        
        // Clear existing annotations from the scene
        for (const [id, annotation] of state.annotations) {
            viewer.scene.annotations.remove(annotation);
        }
        state.annotations.clear();

        // Add new annotations
        for (const annotationData of annotations) {
            const potreeAnnotation = createPotreeAnnotation(annotationData);
            viewer.scene.annotations.add(potreeAnnotation);
        }

        updateAnnotationCount();
        setStatus(`Loaded ${annotations.length} annotations`, 'success');
    } catch (error) {
        console.error('Failed to load annotations:', error);
        setStatus(`Error: ${error.message}`, 'error');
    }
}

/**
 * Save a new annotation
 */
async function saveAnnotation() {
    const title = elements.titleInput.value.trim();
    const description = elements.descriptionInput.value.trim();

    if (!title) {
        setStatus('Title is required', 'error');
        return;
    }

    if (!state.pendingPosition) {
        setStatus('No position selected', 'error');
        return;
    }

    elements.saveButton.disabled = true;
    setStatus('Saving annotation...');

    try {
        const annotationData = await api.createAnnotation({
            x: state.pendingPosition.x,
            y: state.pendingPosition.y,
            z: state.pendingPosition.z,
            title: title,
            description: description,
        });

        // Create and add the Potree annotation
        const potreeAnnotation = createPotreeAnnotation(annotationData);
        viewer.scene.annotations.add(potreeAnnotation);

        updateAnnotationCount();
        hideDialog();
        setStatus('Annotation created successfully', 'success');
    } catch (error) {
        console.error('Failed to save annotation:', error);
        setStatus(`Error: ${error.message}`, 'error');
    } finally {
        elements.saveButton.disabled = false;
    }
}

/**
 * Delete an annotation
 */
async function deleteAnnotation(id) {
    if (!confirm('Are you sure you want to delete this annotation?')) {
        return;
    }

    setStatus('Deleting annotation...');

    try {
        await api.deleteAnnotation(id);

        // Remove from scene
        const annotation = state.annotations.get(id);
        if (annotation) {
            viewer.scene.annotations.remove(annotation);
            state.annotations.delete(id);
        }

        updateAnnotationCount();
        setStatus('Annotation deleted', 'success');
    } catch (error) {
        console.error('Failed to delete annotation:', error);
        setStatus(`Error: ${error.message}`, 'error');
    }
}

/**
 * Toggle annotation adding mode
 */
function toggleAddMode() {
    state.isAddingAnnotation = !state.isAddingAnnotation;
    
    if (state.isAddingAnnotation) {
        elements.addButton.classList.add('active');
        setStatus('Click on the point cloud to add an annotation');
    } else {
        elements.addButton.classList.remove('active');
        setStatus('Ready');
    }
}

/**
 * Handle click on point cloud for adding annotations
 */
function handlePointCloudClick(event) {
    if (!state.isAddingAnnotation) {
        return;
    }

    const mouse = new THREE.Vector2();
    const rect = viewer.renderer.domElement.getBoundingClientRect();
    mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
    mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

    // Get intersection with point cloud
    const intersection = Potree.Utils.getMousePointCloudIntersection(
        mouse,
        viewer.scene.getActiveCamera(),
        viewer,
        viewer.scene.pointclouds,
        { pickClipped: true }
    );

    if (intersection) {
        state.pendingPosition = {
            x: intersection.location.x,
            y: intersection.location.y,
            z: intersection.location.z,
        };
        
        // Exit add mode and show dialog
        toggleAddMode();
        showDialog('Create Annotation');
    } else {
        setStatus('No point detected at click location', 'error');
    }
}

/**
 * Initialize the Potree viewer
 */
function initViewer() {
    window.viewer = new Potree.Viewer(document.getElementById("potree_render_area"));

    viewer.setEDLEnabled(true);
    viewer.setFOV(60);
    viewer.setPointBudget(1_000_000);
    viewer.loadSettingsFromURL();
    viewer.setBackground("gradient");

    viewer.loadGUI(() => {
        viewer.setLanguage('en');
        $("#menu_scene").next().show();
        viewer.toggleSidebar();
    });

    viewer.setDescription(`
        <b>Point Cloud Annotator</b><br>
        <br>
        Click "Add" to create new annotations.<br>
        Hover over annotations to see the delete button.<br>
        All annotations are persisted to the server.
    `);

    // Load the lion point cloud
    Potree.loadPointCloud("/potree/pointclouds/lion_takanawa/cloud.js", "lion", function(e) {
        viewer.scene.addPointCloud(e.pointcloud);
        
        // Set initial camera position
        viewer.scene.view.position.set(4.15, -6.12, 8.54);
        viewer.scene.view.lookAt(new THREE.Vector3(0, -0.098, 4.23));
        
        // Configure point cloud material
        e.pointcloud.material.pointSizeType = Potree.PointSizeType.ADAPTIVE;
        e.pointcloud.material.size = 1;

        // Load existing annotations after point cloud is loaded
        loadAnnotations();
    });

    // Add click handler for adding annotations
    viewer.renderer.domElement.addEventListener('click', handlePointCloudClick);
}

/**
 * Setup event listeners
 */
function setupEventListeners() {
    // Toolbar buttons
    elements.addButton.addEventListener('click', toggleAddMode);
    elements.refreshButton.addEventListener('click', loadAnnotations);

    // Dialog buttons
    elements.saveButton.addEventListener('click', saveAnnotation);
    elements.cancelButton.addEventListener('click', hideDialog);

    // Close dialog on Escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            if (!elements.dialog.classList.contains('hidden')) {
                hideDialog();
            } else if (state.isAddingAnnotation) {
                toggleAddMode();
            }
        }
    });

    // Save on Enter key in title field
    elements.titleInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            saveAnnotation();
        }
    });

    // Close dialog when clicking outside
    elements.dialog.addEventListener('click', (e) => {
        if (e.target === elements.dialog) {
            hideDialog();
        }
    });
}

/**
 * Main initialization
 */
function init() {
    setupEventListeners();
    initViewer();
}

// Start the application
init();
