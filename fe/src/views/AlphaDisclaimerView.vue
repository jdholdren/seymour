<template>
  <div>
    <div class="py-8">
      <h1 class="text-5xl font-bold">Alpha Version Notice</h1>
    </div>

    <div class="bg-primary/10 border-l-4 border-primary p-4 mb-6">
      <div class="flex">
        <div>
          <p class="text-sm text-gray-800">
            <strong>This application is currently in alpha.</strong> You may encounter bugs,
            incomplete features, or unexpected behavior.
          </p>
        </div>
      </div>
    </div>

    <div class="space-y-4 text-gray-700">
      <h2 class="text-xl font-semibold text-gray-900">Found a bug or issue?</h2>

      <p>
        We appreciate your feedback! If you encounter any problems or have suggestions for
        improvement, please report them on our GitHub repository:
      </p>

      <div class="mt-4">
        <a
          href="https://github.com/jdholdren/seymour/issues"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex items-center px-4 py-2 bg-primary text-white rounded-md hover:bg-primary/80 transition-colors"
        >
          <svg class="w-5 h-5 mr-2" fill="currentColor" viewBox="0 0 20 20">
            <path
              fill-rule="evenodd"
              d="M10 0C4.477 0 0 4.484 0 10.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0110 4.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.203 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.942.359.31.678.921.678 1.856 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0020 10.017C20 4.484 15.522 0 10 0z"
              clip-rule="evenodd"
            ></path>
          </svg>
          Report an Issue
        </a>
      </div>

      <h2 class="text-xl font-semibold text-gray-900 mt-6">Thank you!</h2>

      <p>
        Thanks for trying out Seymour in its early stages. Your feedback and patience help us build
        a better RSS reader.
      </p>
    </div>

    <div class="mt-8 pt-6 border-t border-gray-200">
      <router-link to="/timeline" class="text-primary hover:text-primary/80 transition-colors">
        ← Back to Timeline
      </router-link>
    </div>
  </div>
</template>

<script setup>
// No additional logic needed for this static informational page
</script>
