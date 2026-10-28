import { render, screen } from '@testing-library/react';
import React from 'react';

import { LinkButton } from './Button';

describe('LinkButton', () => {
  it.each(['javascript:alert(1)', 'javascript:/*://*/alert(1)', 'data:text/html,<script>alert(1)</script>'])(
    'sanitizes unsafe href %s',
    (href) => {
      render(<LinkButton href={href}>See source</LinkButton>);
      expect(screen.getByRole('link', { name: 'See source' })).toHaveAttribute('href', 'about:blank');
    }
  );

  it.each(['https://grafana.com/docs', '/dashboards', ''])('preserves safe href %s', (href) => {
    render(<LinkButton href={href}>See source</LinkButton>);
    expect(screen.getByRole('link', { name: 'See source' })).toHaveAttribute('href', href);
  });

  it('does not add href when none is provided', () => {
    render(<LinkButton>See source</LinkButton>);
    expect(screen.getByText('See source').closest('a')).not.toHaveAttribute('href');
  });
});
