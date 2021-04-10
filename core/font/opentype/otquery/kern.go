package otquery

/*   Snipped from HarfBuzz fallback shaper:

hb_bool_t
_hb_fallback_shape (hb_shape_plan_t    *shape_plan HB_UNUSED,
		    hb_font_t          *font,
		    hb_buffer_t        *buffer,
		    const hb_feature_t *features HB_UNUSED,
		    unsigned int        num_features HB_UNUSED)
{
  * TODO
   *
   * - Apply fallback kern.
   * - Handle Variation Selectors?
   * - Apply normalization?
   *
   * This will make the fallback shaper into a dumb "TrueType"
   * shaper which many people unfortunately still request.
   *

   hb_codepoint_t space;
   bool has_space = (bool) font->get_nominal_glyph (' ', &space);

   buffer->clear_positions ();

   hb_direction_t direction = buffer->props.direction;
   hb_unicode_funcs_t *unicode = buffer->unicode;
   unsigned int count = buffer->len;
   hb_glyph_info_t *info = buffer->info;
   hb_glyph_position_t *pos = buffer->pos;
   for (unsigned int i = 0; i < count; i++)
   {
	 if (has_space && unicode->is_default_ignorable (info[i].codepoint)) {
	   info[i].codepoint = space;
	   pos[i].x_advance = 0;
	   pos[i].y_advance = 0;
	   continue;
	 }
	 (void) font->get_nominal_glyph (info[i].codepoint, &info[i].codepoint);
	 font->get_glyph_advance_for_direction (info[i].codepoint,
						direction,
						&pos[i].x_advance,
						&pos[i].y_advance);
	 font->subtract_glyph_origin_for_direction (info[i].codepoint,
							direction,
							&pos[i].x_offset,
							&pos[i].y_offset);
   }

   if (HB_DIRECTION_IS_BACKWARD (direction))
	 hb_buffer_reverse (buffer);

   buffer->safe_to_break_all ();

   return true;
 }


*/
