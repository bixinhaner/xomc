var idGlobal = 3;

Vue.component('sort-column', {
    template: `
    <div class="showHideItem" id="showOrHideItem">
        <div class="flex-ctn" id="sortAndShowColumnBoxCls" style="height:540px;flex-direction: row;border-bottom: 1px solid #e9e9e9;">
            <div style="padding: 0px 0 0 10px;height:100%;flex:3;overflow:auto;">
                <div class="select-all-cls">
                    <span style="margin: 0;font-weight: bold;">{{local.selectColumn}}</span>
                </div>
                <div class="select-all-cls" style="padding-left: 10px;">
                    <el-checkbox :indeterminate="!colAll" v-model="colAll" @change="colAllChange"></el-checkbox> 
                    <span>{{local.selectAll}}</span>
                </div>
                <div>
                    <div class="select-all-cls">
                        <i :class="{'el-icon':true,'el-icon-close1':expanded.device,'el-icon-open':!expanded.device}" @click="expanded.device = !expanded.device"></i>
                        <el-checkbox :indeterminate="form.device.length<deviceCol.length" v-model="deviceAll" @change="deviceAllChange"></el-checkbox> 
                        <span>{{local.deviceInfo}}</span>
                    </div>
                    <el-checkbox-group v-show="expanded.device" class="col-group" v-model="form.device">
                        <el-checkbox v-for="item in deviceCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
                    </el-checkbox-group>
                </div>
                <div>
                    <div class="select-all-cls">
                        <i :class="{'el-icon':true,'el-icon-close1':expanded.cell,'el-icon-open':!expanded.cell}" @click="expanded.cell = !expanded.cell"></i>
                        <el-checkbox :indeterminate="form.cell.length<cellCol.length" v-model="cellAll" @change="cellAllChange"></el-checkbox> 
                        <span>{{local.cellInfo}}</span>
                    </div>
                    <el-checkbox-group v-show="expanded.cell" class="col-group" v-model="form.cell">
                        <el-checkbox v-for="item in cellCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
                    </el-checkbox-group>
                </div>
                <div>
                    <div class="select-all-cls">
                        <i :class="{'el-icon':true,'el-icon-close1':expanded.status,'el-icon-open':!expanded.status}" @click="expanded.status = !expanded.status"></i>
                        <el-checkbox :indeterminate="form.status.length<statusCol.length" v-model="statusAll" @change="statusAllChange"></el-checkbox> 
                        <span>{{local.status}}</span>
                    </div>
                    <el-checkbox-group v-show="expanded.status" class="col-group" v-model="form.status">
                        <el-checkbox v-for="item in statusCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
                    </el-checkbox-group>
                </div>
                <div>
                    <div class="select-all-cls">
                        <i :class="{'el-icon':true,'el-icon-close1':expanded.network,'el-icon-open':!expanded.network}" @click="expanded.network = !expanded.network"></i>
                        <el-checkbox :indeterminate="form.network.length<networkCol.length" v-model="networkAll" @change="networkAllChange"></el-checkbox> 
                        <span>{{local.network}}</span>
                    </div>
                    <el-checkbox-group v-show="expanded.network" class="col-group" v-model="form.network">
                        <el-checkbox v-for="item in networkCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
                    </el-checkbox-group>
                </div>
                <div>
                    <div class="select-all-cls">
                        <i :class="{'el-icon':true,'el-icon-close1':expanded.location,'el-icon-open':!expanded.location}" @click="expanded.location = !expanded.location"></i>
                        <el-checkbox :indeterminate="form.location.length<locationCol.length && form.location.length>0" v-model="locationAll" @change="locationAllChange"></el-checkbox> 
                        <span>{{local.position}}</span>
                    </div>
                    <el-checkbox-group v-show="expanded.location" class="col-group" v-model="form.location">
                        <el-checkbox v-for="item in locationCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}">{{item.label}}</el-checkbox>
                    </el-checkbox-group>
                </div>
                <div>
                    <div class="select-all-cls">
                        <i :class="{'el-icon':true,'el-icon-close1':expanded.satellite,'el-icon-open':!expanded.satellite}" @click="expanded.satellite = !expanded.satellite"></i>
                        <el-checkbox v-model="satelliteAll" @change="satelliteAllChange"></el-checkbox> 
                        <span>{{local.satellite}}</span>
                    </div>
                    <el-checkbox-group v-show="expanded.satellite" class="col-group" v-model="form.satellite">
                        <el-checkbox v-for="item in satelliteCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}">{{item.label}}</el-checkbox>
                    </el-checkbox-group>
                </div>
            </div>
            
            <div style="border-left: 1px solid #e9e9e9;height:100%;flex:1;">
                <div class="select-all-cls" style="margin: 10px; padding: 0px; border-bottom: 1px solid #e9e9e9;">
                    <el-checkbox v-if="false" :indeterminate="!dragAll" v-model="dragAll" @change="dragAllChange"></el-checkbox> 
                    <span v-if="false">{{local.selectAll}}</span>
                    <span style="padding-bottom: 10px;margin-left: 0px;font-weight: bold;">{{local.collumSort}}</span>
                </div>
                <el-checkbox-group v-model="dragCol">
                    <draggable
                        class="list-group"
                        v-model="columns"
                        v-bind="dragOptions">
                        <transition-group type="transition" :name="!drag? 'flip-list':null">
                            <div v-for="(col,idx) in columns" :key="col.field" class="list-group-item" v-if="showCols.includes(col.field)">
                                <span class="el-checkbox__label" style="border-bottom: 1px dashed #e9e9e9;min-width: 260px;color: #606266;">
                                    {{col.label}}

                                    <i v-if="!col.disabled" class="el-icon el-icon-close sort-item-op" style="zoom: 0.6;float: right; margin-top: 6px;" @click="clickColumnLabel(col.label)"></i>
                                </span>
                                <el-checkbox v-if="false" :label="col.field" :key="col.field" :disabled="col.disabled">{{col.label}} </el-checkbox>
                            </div>
                        </transition-group>
                    </draggable>
                </el-checkbox-group>
            </div>
        </div>
        <div class="windowButtonGroup" style="float:none !important;padding:20px 0 20px 30px;position:relative;z-index:321;">			
            <a class="linkbutton linkbutton_trend" @click="ColumnConfigEn()"><span>{{local.ok}}</span></a>
            <a class="linkbutton linkbutton_nowanna" @click="close"><span>{{local.cancel}}</span></a>
        </div>
    </div>
    `,
    props: {
        local: {
            type: Object,
            default() {
                return {
                    selectColumn: '',
                    selectAll: '',
                    deviceInfo: '',
                    cellInfo: '',
                    status: '',
                    network: '',
                    position: '',
                    satellite: '',
                    collumSort: '',
                    ok: '',
                    cancel: '',
                    enbId: '',
                    hostName: '',
                    cellId: '',
                    eci: '',
                    pci: '',
                    plmnId: '',
                    tac: '',
                    signment: '',
                    subframe: '',
                    rootIndex: '',
                    bandwidth: '',
                    earfcn: '',
                    netModel: '',
                    txpower: '',
                    contactNum: '',
                    activeStatus: '',
                    mmeStatus: '',
                    rfStatus: '',
                    pmStatus: '',
                    halob: '',
                    synStatus: '',
                    validity: '',
                    lockStatus: '',
                    ueCount: '',
                    euCount: '',
                    ruCount: '',
                    cpeConn: '',
                    wanSpeed: '',
                    serviceStatus: '',
                    mmePool: '',
                    cellIp: '',
                    ipsecAddr: '',
                    sn: '',
                    product: '',
                    productName: '',
                    module: '',
                    software: '',
                    firmware: '',
                    duration: '',
                    upTime: '',
                    firstTime: '',
                    lastTime: '',
                    mac: '',
                    gpsVersion: '',
                    group: '',
                    longitude: '',
                    latitude: '',
                    height: '',
                    mechDowntilt: '',
                    elecDowntilt: '',
                    beamWidth: '',
                    azimuth: '',
                    installAddr: '',
                    rate: ''
                }
            }
        },
        ctx: {
            type: String,
            default: ''
        }
    },
    data() {

		var ignores = [];

		if("${isSuperAdmin}" != '1'){
			ignores.push('IPSEC_ADDR');
			ignores.push('mmepool_ipsec_addr');
		}
		/* 目前只有Amara支持ups，后面后端会调整逻辑 */
		/*if(siteIdShow != 'true'){
			ignores.push('site_id');
		}*/
		/* 以下字段受此参数('${enbAdditionalColShow}')控制显示或隐藏 */
		if(!supportTopoSite && '${enbAdditionalColShow}' != 'true') {
			ignores.push('sub_station_name');
		}

		if('${enbAdditionalColShow}' != 'true'){
			ignores.push('site_id');
			
			ignores.push('install_address');
			ignores.push('circuit_ref');
			ignores.push('circuit_jo');
			ignores.push('service_status');
			ignores.push('rom');
			ignores.push('contact_number');
		}
		if (isCloud != 'true' && false){
			ignores.push('available_rate');
		}
		//cloud版的支持halob 
		if( isSupportHalob != 'true'){
			ignores.push('halob_flag');
		}
		if(writableMap["CODE_ENB_EXPIRY_DATE"] == undefined){
			ignores.push('validity');
			//ignores.push('lock_status');
		}

		var cellCol= [
            	{code: 'enbId', label: this.local.enbId, width: 100},
				{code: 'host_name', label: this.local.hostName,disabled: true},
                {code: 'cellId', label: this.local.hostName, width: 100},
				{code: 'CELL_IDENTITY', label: 'ECI',disabled: true},
				{code: 'PHYCELLID', label: this.local.pci,disabled: true},
				{code: 'plmnid', label: this.local.plmnId},
				{code: 'tac', label: this.local.tac},
				{code: 'signment', label: this.local.signment},
				{code: 'specialSubframe', label: this.local.subframe},
				{code: 'rootIndex', label: this.local.rootIndex},
				{code: 'site_id', label: siteIdLabel},
				{code: 'bandwidth', label: this.local.bandwidth},
				{code: 'EARFCNDLINUSE', label: this.local.earfcn},
				{code: 'network_model', label: this.local.netModel},
				{code: 'tx_power',label: this.local.txpower},
				{code: 'circuit_ref', label: 'Circuit Ref.'},
                {code: 'circuit_jo', label: 'Circuit J & O'},
                {code: 'contact_number', label: this.local.contactNum}
			],
			statusCol= [
				{code: 'op_state', label: this.local.activeStatus,disabled: true},
				{code: 'mme_status', label: this.local.mmeStatus,disabled: true},
				{code: 'rf_status', label: this.local.rfStatus,disabled: true},
				{code: 'pm_report_status', label: this.local.pmStatus},
				{code: 'halob_flag', label: 'HaloX'},
				{code: 'synStatus', label: this.local.synStatus},
				{code: 'validity', label: this.local.validity},
				{code: 'lock_status', label: this.local.lockStatus},
				{code: 'ue_count', label: this.local.ueCount,disabled: true},
				{code: 'euCountStr', label: this.local.euCount},
				{code: 'ruCountStr', label: this.local.ruCount},
				{code: 'cpe_connect', label: this.local.cpeConn,disabled: true},
				{code: 'wanSpeed', label: this.local.wanSpeed},
				{code: 'service_status', label: this.local.serviceStatus}
			],
			networkCol= [
				{code: 'mmepool_ipsec_addr', label: this.local.mmePool},
				{code: 'cell_ip', label: this.local.cellIp,disabled: true},
				{code: 'IPSEC_ADDR', label: this.local.ipsecAddr}
			],
			deviceCol= [
				{code: 'serial_number', label: this.local.sn,disabled: true},
				{code: 'product', label: this.local.product,disabled: true},
				{code: 'product_name', label: this.local.productName},
				{code: 'module_type', label: this.local.module,disabled: true},
				{code: 'software_version', label: this.local.software,disabled: true},
				{code: 'firmware_version', label: this.local.firmware},
				{code: 'online_duration', label: this.local.duration},
				{code: 'up_time', label: this.local.upTime},
				{code: 'first_online_time', label: this.local.firstTime},
				{code: 'LASTINFORMTIME', label: this.local.lastTime},
				{code: 'mac_address', label: this.local.mac,disabled: true},
				{code: 'gps_version', label: this.local.gpsVersion},
				{code: 'group_name', label: this.local.group,disabled: true},
				{code: 'sub_station_name', label: siteNameLabel},
				{code: 'rom', label: 'Rom'}
			],
			locationCol= [
				{code: 'gps_longitude', label: this.local.longitude},
				{code: 'gps_latitude', label: this.local.latitude},
				{code: 'gps_height', label: this.local.height},
				{code: 'mechanical_downtilt', label: this.local.mechDowntilt},
				{code: 'electronic_downtilt', label: this.local.elecDowntilt},
				{code: 'vertical_3dB_beam_width', label: this.local.beamWidth},
				{code: 'horizontal_azimuth', label: this.local.azimuth},
				{code: 'install_address', label: this.local.installAddr}
			];

		return {
			dragCol: [
				// 'serial_number',
				// 'host_name',
				'rf_status',
				'op_state',
				'CELL_IDENTITY',
				'PHYCELLID',
				'mme_status',
				'ue_count',
				'cpe_connect',
				'cell_ip',
				'mac_address',
				'product',
				'module_type',
				'software_version'
			],
			drag: false,
			columns: [
				{field: 'enbId', label: this.local.enbId, width: 100},
				{field: 'cellId', label: this.local.cellId, width: 100},
				{field: 'rf_status', label: this.local.rfStatus,sortable: true,disabled: true, width: 150},
				{field: 'op_state', label: this.local.activeStatus,sortable: true,disabled: true, width: 120},
				{field: 'CELL_IDENTITY', label: 'ECI',sortable: true,disabled: true, width: 100},
				{field: 'PHYCELLID', label: this.local.pci,sortable: true,disabled: true, width: 70},
				{field: 'mme_status', label: this.local.mmeStatus,disabled: true, width: 120},
				{field: 'plmnid', label: this.local.plmnId, width: 70},
				{field: 'bandwidth', label: this.local.bandwidth, width: 80},
				{field: 'ue_count', label: this.local.ueCount,sortable: true,disabled: true, width: 80},
				
				{field: 'euCountStr', label: this.local.euCount,sortable: true, width: 80},
				{field: 'ruCountStr', label: this.local.ruCount,sortable: true, width: 80},
				
				{field: 'cpe_connect', label: this.local.cpeConn,disabled: true, width: 100},
				{field: 'wanSpeed', label: this.local.wanSpeed, width: 180},
				{field: 'cell_ip', label: this.local.cellIp,sortable: true,disabled: true, width: 120},
				{field: 'mac_address', label: this.local.mac,sortable: true,disabled: true, width: 130},
				{field: 'product', label: this.local.product,sortable: true,disabled: true, width: 110},
				{field: 'product_name', label: this.local.productName,sortable: true, width: 120},
				{field: 'module_type', label: this.local.module,sortable: true,disabled: true, width: 120},
				{field: 'software_version', label: this.local.software,sortable: true,disabled: true, width: 140},
				{field: 'group_name', label: this.local.group,sortable: true,disabled: true, width: 130},
				{field: 'IPSEC_ADDR', label: this.local.ipsecAddr, width: 170},
				{field: 'mmepool_ipsec_addr', label: this.local.mmePool, width: 120},
				{field: 'site_id', label: siteIdLabel,sortable: true, width: 80},
				
				{field: 'sub_station_name', label: siteNameLabel, width: 120},
                {field: 'install_address', label: this.local.installAddr, width: 200},
                {field: 'circuit_ref', label: 'Circuit Ref.', width: 120},
                {field: 'circuit_jo', label: 'Circuit J & O', width: 120},
                {field: 'service_status', label: this.local.serviceStatus,width: 120},
                {field: 'rom', label: 'Rom', width: 120},
                {field: 'contact_number', label: this.local.contactNum, width: 120},
                
				{field: 'EARFCNDLINUSE', label: this.local.earfcn, width: 130},
				{field: 'synStatus', label: this.local.synStatus,sortable: true, width: 160},
				{field: 'pm_report_status', label: this.local.pmStatus,sortable: true, width: 135},
				{field: 'gps_satellite_count', label: this.local.satellite,sortable: true, width: 85},
				{field: 'online_duration', label: this.local.duration,sortable: true, width: 120},
				{field: 'up_time', label: this.local.upTime,sortable: true, width: 120},
				{field: 'first_online_time', label: this.local.firstTime,sortable: true, width: 140},
				{field: 'LASTINFORMTIME', label: this.local.lastTime,sortable: true, width: 140},
				{field: 'network_model', label: this.local.netModel,sortable: true, width: 120},
				{field: 'firmware_version', label: this.local.firmware,sortable: true, width: 135},
				{field: 'gps_version', label: this.local.gpsVersion, width: 120},
				{field: 'available_rate', label: this.local.rate, width: 70},
				{field: 'halob_flag', label: this.local.halob,sortable: true, width: 100},
				{field: 'tac', label: this.local.tac, width: 80},
				{field: 'signment', label: this.local.signment, width: 80},
				{field: 'specialSubframe', label: this.local.subframe, width: 80},
				{field: 'rootIndex', label: this.local.rootIndex, width: 80},
				{field: 'gps_longitude', label: this.local.longitude, width: 100},
				{field: 'gps_latitude', label: this.local.latitude, width: 90},
				{field: 'gps_height', label: this.local.height, width: 70},

				{field: 'mechanical_downtilt', label: this.local.mechDowntilt, width: 135},
				{field: 'electronic_downtilt', label: this.local.elecDowntilt, width: 135},
				{field: 'vertical_3dB_beam_width', label: this.local.beamWidth, width: 160},
				{field: 'horizontal_azimuth', label: this.local.azimuth, width: 135},

				{field: 'validity', label: this.local.validity, width: 125},
				{field: 'lock_status', label: this.local.lockStatus, width: 90},
				{field: 'tx_power', label: this.local.txpower, width: 80}
			],
			cellCol: cellCol.filter(function(item){ return !ignores.includes(item.code);}),
			statusCol: statusCol.filter(function(item){ return !ignores.includes(item.code);}),
			networkCol: networkCol.filter(function(item){ return !ignores.includes(item.code);}),
			deviceCol: deviceCol.filter(function(item){ return !ignores.includes(item.code);}),
			locationCol: locationCol.filter(function(item){ return !ignores.includes(item.code);}),			
			satelliteCol: [
				{code: 'gps_satellite_count', label: this.local.satellite}
			],
			form: {
				device: ['serial_number','product','module_type','software_version','mac_address','group_name'],
				cell: ['host_name','CELL_IDENTITY','PHYCELLID'],
				status: ['op_state','mme_status','rf_status','ue_count','cpe_connect'],
				network: ['cell_ip'],
				location: [],
				satellite: []
			},
			expanded: {
				device: true,
				cell: true,
				status: true,
				network: true,
				location: true,
				satellite: true
			}
		};
	},
	computed: {
		dragOptions() {

			return {
				animation: 200,
				group: 'description',
				disabled: false,
				ghostClass: 'ghost'
			};
		},
		dragAll() {
			var vm = this;
			return vm.dragCol.length == vm.columns.length;
		},
		colAll() {
			var vm = this;
			return vm.deviceAll && vm.cellAll && vm.statusAll && vm.locationAll && vm.satelliteAll;
		},
		deviceAll() {
			var vm = this;
			return vm.deviceCol.length == vm.form.device.length;
		},
		cellAll() {
			var vm = this;
			return vm.cellCol.length == vm.form.cell.length;
		},
		statusAll() {
			var vm = this;
			return vm.statusCol.length == vm.form.status.length;
		},
		networkAll() {
			var vm = this;
			return vm.networkCol.length == vm.form.network.length;
		},
		locationAll() {
			var vm = this;
			return vm.locationCol.length == vm.form.location.length;
		},
		satelliteAll() {
			var vm = this;
			return vm.satelliteCol.length == vm.form.satellite.length;
		},
		showCols() {
			var vm = this;

			return vm.form.device.concat(vm.form.cell).concat(vm.form.status).concat(vm.form.network).concat(vm.form.location).concat(vm.form.satellite);
			//return vm.dragCol;
		}
	},
	methods: {
		clickColumnLabel(label) {
			$('.el-checkbox__label:contains('+label+')').click();
		},
		dragAllChange(val) {
			var vm = this,
				fields = vm.columns.map(function(item){
					return item.field;
				}),
				filters = vm.columns.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.field;
				});
			
			vm.dragCol = val?fields:filters;
		},
		colAllChange(val) {
			var vm = this;

			vm.deviceAllChange(val);
			vm.cellAllChange(val);
			vm.statusAllChange(val);
			vm.networkAllChange(val);
			vm.locationAllChange(val);
			vm.satelliteAllChange(val);
		},
		deviceAllChange(val) {
			var vm = this,
				fields = vm.deviceCol.map(function(item){
					return item.code;
				}),
				filters = vm.deviceCol.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.code;
				});
			
			vm.form.device = val?fields:filters;
		},
		cellAllChange(val) {
			var vm = this,
				fields = vm.cellCol.map(function(item){
					return item.code;
				}),
				filters = vm.cellCol.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.code;
				});
			
			vm.form.cell = val?fields:filters;
		},
		statusAllChange(val) {
			var vm = this,
				fields = vm.statusCol.map(function(item){
					return item.code;
				}),
				filters = vm.statusCol.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.code;
				});
			
			vm.form.status = val?fields:filters;
		},
		networkAllChange(val) {
			var vm = this,
				fields = vm.networkCol.map(function(item){
					return item.code;
				}),
				filters = vm.networkCol.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.code;
				});
			
			vm.form.network = val?fields:filters;
		},
		locationAllChange(val) {
			var vm = this,
				fields = vm.locationCol.map(function(item){
					return item.code;
				}),
				filters = vm.locationCol.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.code;
				});
			
			vm.form.location = val?fields:filters;
		},
		satelliteAllChange(val) {
			var vm = this,
				fields = vm.satelliteCol.map(function(item){
					return item.code;
				}),
				filters = vm.satelliteCol.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.code;
				});
			
			vm.form.satellite = val?fields:filters;
		},
		close() {
			$(".showHideItem").slideUp(500);
		},
		ColumnConfigEn() {
			var vm = this,
				url = vm.ctx + '/cell/cpeinfos/cellColumnConfig.action',
				sortCol = [];
				
			vm.columns.map(function(item){
				sortCol.push(item.field);
			});

			vm.configColumn();

			url = vm.ctx + '/system/column/setting/insert.action';
			// 保存显示列
			var params = {
					pageName: '1',
					showColumn: vm.showCols.join(','),
					sortColumn: sortCol.join(',')
				};

			axios.post(url, params).then(function(res){
				enbvm.columns.splice(0, enbvm.columns.length);
				vm.columns.map(function(col){
					enbvm.columns.push(Object.assign({},col));
				});

				var tbStates = enbvm.$refs.list.$refs.ctableInner.store.states,
					sortCodes = sortCol,
					storeCols = tbStates.columns;

				tbStates._columns = storeCols.sort(function(n, m) {
					var idxn = sortCodes.indexOf(n.property)==-1?100:sortCodes.indexOf(n.property),
						idxm = sortCodes.indexOf(m.property)==-1?100:sortCodes.indexOf(m.property);

					if( [undefined,'enb_operation','connection_status','alarm','serial_number','host_name'].includes(n.property) ) {
						idxn = 0;
					}
					if( [undefined,'enb_operation','connection_status','alarm','serial_number','host_name'].includes(m.property) ) {
						idxm = 0;
					}

					return idxn - idxm;
				});

				enbvm.$refs.list.$refs.ctableInner.store.updateColumns();
				$('#cellInfo').css('width','99.9%');
				setTimeout(function(){
					$('#cellInfo').css('width','100%');
				},2000);
			}).catch(function(){});

			vm.close();
		},
		configColumn() {
			var vm = this,
				eNodeB_column = enbvm.getAllDefaultCols();

			var columns = eNodeB_column.filter(function(item){
				return vm.showCols.includes(item.prop) || ['oepr','enb_operation','connection_status','alarm','serial_number','host_name'].includes(item.prop);
			});

			if(enableCheckbox){
				columns.unshift({field:'oepr',checkbox:true,width:50});
			}

			enbvm.showProps = vm.showCols;
		},

		initSort() {
			var vm = this,
				map = {
					device: vm.deviceCol.map(function(item){ return item.code;}),
					cell: vm.cellCol.map(function(item){ return item.code;}),
					status: vm.statusCol.map(function(item){ return item.code;}),
					network: vm.networkCol.map(function(item){ return item.code;}),
					location: vm.locationCol.map(function(item){ return item.code;}),
					satellite: vm.satelliteCol.map(function(item){ return item.code;})

				},
				dragCodes = vm.columns.map(function(item){ return item.field}),
				showCols = enbvm.showProps;

			showCols.map(function(col){
				['device','cell','status','network','location','satellite'].map(function(code){
					if(map[code].includes(col) && !vm.form[code].includes(col)) vm.form[code].push(col);
				});
				// 初始化默认选中项
				if(dragCodes.includes(col) && !vm.dragCol.includes(col)) vm.dragCol.push(col);
			});
			// 假设为排序 --->
			var sortCodes = enbvm.sortColumns,
				sortList = vm.columns;

			vm.columns = sortList.sort(function(n, m){
				var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
					idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

				return idxn - idxm;
			});
			// <--- 排序

			vm.configColumn();
		}
	},
	created() {
		
	},
	mounted() {
		//this.ColumnConfigEn();
	}
})