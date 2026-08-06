from pathlib import Path
import sys
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "python-side"))

from atm.layers import application  # noqa: E402


class ComandoTests(unittest.TestCase):
    """Construccion del texto de peticion que atraviesa Presentacion."""

    def test_comando_login(self):
        self.assertEqual(application.comando_login("4111111111111111", "1234"), "LOGIN|4111111111111111|1234")

    def test_comando_retiro_formatea_dos_decimales(self):
        self.assertEqual(application.comando_retiro(100), "WITHDRAW|100.00")

    def test_comando_logout(self):
        self.assertEqual(application.comando_logout(), "LOGOUT")


class ParseRespuestaTests(unittest.TestCase):
    """Interpretacion del texto plano recibido del banco."""

    def test_login_ok(self):
        respuesta = application.parse_respuesta("LOGIN_OK|Autenticacion exitosa")
        self.assertEqual(respuesta.action, "LOGIN_OK")
        self.assertEqual(respuesta.message, "Autenticacion exitosa")

    def test_withdraw_ok_parsea_montos(self):
        respuesta = application.parse_respuesta("WITHDRAW_OK|100.00|400.00")
        self.assertEqual(respuesta.amount, 100.00)
        self.assertEqual(respuesta.balance, 400.00)

    def test_withdraw_error(self):
        respuesta = application.parse_respuesta("WITHDRAW_ERROR|Fondos insuficientes")
        self.assertEqual(respuesta.message, "Fondos insuficientes")

    def test_texto_malformado_lanza_value_error(self):
        with self.assertRaises(ValueError):
            application.parse_respuesta("LOGIN_OK|falta|un|campo|de|mas")

    def test_accion_desconocida_lanza_value_error(self):
        with self.assertRaises(ValueError):
            application.parse_respuesta("QUIEN_SABE|que es esto")

    def test_withdraw_ok_con_montos_no_numericos_lanza_value_error(self):
        with self.assertRaises(ValueError):
            application.parse_respuesta("WITHDRAW_OK|mucho|poco")


if __name__ == "__main__":
    unittest.main()
