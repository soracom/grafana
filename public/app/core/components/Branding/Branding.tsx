import { css, cx } from '@emotion/css';
import { FC, useState } from 'react';

import { colorManipulator } from '@grafana/data';
import { useTheme2 } from '@grafana/ui';

export interface BrandComponentProps {
  operatorId?: string;
  className?: string;
  children?: JSX.Element | JSX.Element[];
}

export const LoginLogo: FC<BrandComponentProps & { logo?: string }> = ({ className, logo }) => {
  return <img className={className} src={`${logo ? logo : 'public/img/lagoon-logo-cl.svg'}`} alt="Lagoon" />;
};

const LoginBackground: FC<BrandComponentProps> = ({ className, children }) => {
  const theme = useTheme2();

  const background = css({
    '&:before': {
      content: '""',
      position: 'fixed',
      left: 0,
      right: 0,
      bottom: 0,
      top: 0,
      background: `url(public/img/g8_login_${theme.isDark ? 'dark' : 'light'}.svg)`,
      backgroundPosition: 'top center',
      backgroundSize: 'auto',
      backgroundRepeat: 'no-repeat',

      opacity: 0,
      transition: 'opacity 3s ease-in-out',

      [theme.breakpoints.up('md')]: {
        backgroundPosition: 'center',
        backgroundSize: 'cover',
      },
    },
  });

  return <div className={cx(background, className)}>{children}</div>;
};

async function imageExists(imageURL: string) {
  const response = await fetch(imageURL, {
    method: 'HEAD',
  });

  if (response.status === 200) {
    return true;
  } else {
    return false;
  }
}

const MenuLogo: FC<BrandComponentProps> = ({ operatorId, className }) => {
  const [isChecking, setIsChecking] = useState(false);
  const [imageURL, setImageURL] = useState('');
  const lagoonIcon = 'public/img/lagoon-logo-cl.svg';

  if (!isChecking && imageURL === '' && operatorId && operatorId.endsWith('-PRO')) {
    setIsChecking(true);
    const url = 'https://soracom-customer-images.s3.amazonaws.com/lagoon/' + operatorId + '/logo';
    imageExists(url)
      .then((result) => {
        if (result) {
          setImageURL(url);
        } else {
          setImageURL(lagoonIcon);
        }
      })
      .finally(() => {
        setIsChecking(false);
      });
  } else if (operatorId && !operatorId.endsWith('-PRO')) {
    return <img className={className} src={lagoonIcon} alt="Lagoon" />;
  }

  if (imageURL === '') {
    return null;
  }
  return <img className={className} src={imageURL} alt="Lagoon" />;
};

const LoginBoxBackground = () => {
  const theme = useTheme2();
  return css({
    background: colorManipulator.alpha(theme.colors.background.primary, 0.7),
    backgroundSize: 'cover',
  });
};

export class Branding {
  static LoginLogo = LoginLogo;
  static LoginBackground = LoginBackground;
  static MenuLogo = MenuLogo;
  static LoginBoxBackground = LoginBoxBackground;
  static AppTitle = 'SORACOM Lagoon';
  static LoginTitle = 'Welcome to SORACOM Lagoon';
  static HideEdition = false;
  static GetLoginSubTitle = (): null | string => {
    return null;
  };
}
