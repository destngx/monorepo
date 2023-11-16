import { isAuthenticated } from './global-store';

describe('libsCloudinaryPhotosAppGlobalStore', () => {
  it('should work', () => {
    expect(isAuthenticated).toBeDefined();
  });
});
