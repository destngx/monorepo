export const baseConfig = () => ({
  port: parseInt(process.env.PORT) || 3000,
  pokemonService: {
    apiKey: process.env.POKEMEON_KEY,
  },
});
