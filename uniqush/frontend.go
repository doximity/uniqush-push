/*
 *  Uniqush by Nan Deng
 *  Copyright (C) 2010 Nan Deng
 *
 *  This software is free software; you can redistribute it and/or
 *  modify it under the terms of the GNU Lesser General Public
 *  License as published by the Free Software Foundation; either
 *  version 3.0 of the License, or (at your option) any later version.
 *
 *  This software is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 *  Lesser General Public License for more details.
 *
 *  You should have received a copy of the GNU Lesser General Public
 *  License along with this software; if not, write to the Free Software
 *  Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  USA
 *
 *  Nan Deng <monnand@gmail.com>
 *
 */

package uniqush

import (
    "log"
)

type UniqushFrontEnd interface {
    SetChannel(ch chan *Request)
    SetLogger(logger *log.Logger)

    // writer will be used to report real-time event
    SetEventWriter(writer *EventWriter)

    SetStopChannel(ch chan bool)
    Run()
}
