# embedding.service.py


from flask import Flask, request, jsonify
from sentence_transformers import SentenceTransformer

# Load a pretrained model.
# 'all-MiniLM-L6-v2'
print("Loading sentence transformer model...")
model = SentenceTransformer('all-MiniLM-L6-v2')
print("Model loaded successfully.")

app = Flask(__name__)


@app.route('/embed', methods=['POST'])
def embed():
    try:
        data = request.get_json()
        if not data or 'text' not in data:
            return jsonify({"error": "Request body must be JSON with a 'text' key"}), 400

        text_to_embed = data['text']

        embedding = model.encode(text_to_embed).tolist()

        return jsonify({"embedding": embedding})

    except Exception as e:
        print(f"An error occurred: {e}")
        return jsonify({"error": "Failed to generate embedding"}), 500


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001)
