import { Card, CardActionArea, CardContent, Typography, Box, Tooltip } from '@mui/material';
import { useNavigate } from 'react-router-dom';

interface BehaviorCardProps {
  title: string;
  description: string;
  icon: React.ReactNode;
  path: string;
}

export default function BehaviorCard({ title, description, icon, path }: BehaviorCardProps) {
  const navigate = useNavigate();

  const handleClick = () => {
    navigate(path);
  };

  return (
    <Tooltip title={description} arrow placement="top">
      <Card
        sx={{
          width: 280,
          height: 200,
          transition: 'transform 0.2s, box-shadow 0.2s',
          '&:hover': {
            transform: 'translateY(-4px)',
            boxShadow: 6,
          },
        }}
      >
        <CardActionArea
          onClick={handleClick}
          sx={{
            height: '100%',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'center',
            alignItems: 'center',
          }}
        >
          <CardContent
            sx={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              textAlign: 'center',
              gap: 2,
            }}
          >
            <Box
              sx={{
                fontSize: '4rem',
                color: 'primary.main',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              {icon}
            </Box>
            <Typography variant="h5" component="h2">
              {title}
            </Typography>
          </CardContent>
        </CardActionArea>
      </Card>
    </Tooltip>
  );
}
