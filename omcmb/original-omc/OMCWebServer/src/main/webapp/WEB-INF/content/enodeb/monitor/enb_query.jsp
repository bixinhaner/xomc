<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<!-- 显示隐藏列 -->
<div class="showHideItem" id="showOrHideItem"></div>
<div v-if="isExisted && writableMap['CODE_ENB_TR069_MSG_EXCHANGE'] == true" class="fixed-right-msg">
	<div v-if="msgExtend" style="margin-right: 20px;line-height:26px;">
		<span style="padding: 0px 10px;">{{collectSn}}</span>
		<span style="padding: 0px 5px;" v-if="taskTime==''">
			<i class="el-icon el-icon-status-yes" style="font-size: 12px;"></i>
			<%=rb.getString("ChengGong")%>
		</span>
		<span v-if="taskTime!=''" style="display: inline-block;padding: 2px 30px;background: #4d84ff;border-radius: 2px;margin: 0px 5px 2px 5px;"></span>
		<span v-if="taskTime!=''" style="border: 1px solid #e3e3e3;border-radius: 3px;padding: 2px 4px;">
			<span style="cursor: pointer;" @click="stopCollect">
				<i style="padding: 4px;background: red;height: 0px;display: inline-block;border-radius: 3px;"></i>
				<%=rb.getString("TingZhi")%>
			</span>
			<span style="margin-left: 5px;">
				{{taskTime}}
			</span>
		</span>
		<span style="margin-left: 20px;">
			<a class="collect-bt" @click="viewMsg"><%=rb.getString("ChaKan")%></a>
			<a class="collect-bt" @click="downloadMsg"><%=rb.getString("XiaZai")%></a>
			<a class="collect-bt" @click="clearMsg"><%=rb.getString("QingChu")%></a>
		</span>
	</div>
	<div @click="msgExtend = !msgExtend" class="foldBtnCls">
		<span v-if="msgExtend" class="el-icon el-icon-common-query-up"></span>
		<span v-if="!msgExtend" class="el-icon el-icon-common-query-down"></span>
	</div>
</div>
<!-- progress -->
<div v-if="!progressHide" class="fixed-right-msg" :style="{right: msgExtend? (taskTime==''?'600px':'700px'):'150px', display: 'flex', 'align-items': 'baseline', padding: '5px 0 5px 15px'}">
	<%=rb.getString("DaoChuJinDu")%>: <el-progress :percentage="percentage" style="width:120px;margin-left: 5px;" color="#f56c6c"></el-progress>
</div>

<div id="tableHeadQuery" class="tableHeadQueryBoxCls" style="padding-right:120px;">
	<div class="headQueryBox headQueryBoxInputWidthCls">
		<div class="queryGroup">
			<el-input v-model="search_text" @keyup.enter.native="query" @focus="queryInputFocus" @blur="queryInputBlur" :placeholder='placeholderText' style="width:340px;"></el-input>
			<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
		</div>
	</div>
	<div v-for="(item,index) in advancedQueryItemList">
		<div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
			<el-popfilter
				:label='item.label'
				v-model="item.checkedItemList"
				:list="item.options"
				:visible.sync="item.isShow"
				:closable="true"
				@check-change="advanceQuery(item.type,item.value,item.checkedItemList)"
				@close="checkItemDel(item)">
			</el-popfilter>
		</div>
		<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
			<el-popfilter
				type="single"
				:label='item.label'
				v-model="item.selectVal"
				:list="item.options"
				:visible.sync="item.isShow"
				:closable="true"
				@check-change="advanceQuery(item.type,item.value,item.selectVal)"
				@close="checkItemDel(item)">
			</el-popfilter>
		</div>
		<div v-if="item.type == 'filter'" style="margin-right:10px;">
			<div class="advancedQueryItemBox" style="background: #FFF;">
				<el-popover :ref="'popover-'+item.value" trigger="click" placement="bottom-start"  @show="checkPopoverShow(item)" @hide="checkPopoverHide(item,'')">
					<div class="checkPopoverBoxCls">
						<el-checkbox-group v-model="item.checkedItemList" @change="handleCheckedChange(item)">
							<el-checkbox v-for=" items in item.options" :key="items.value" :label="items.value">{{items.label}}</el-checkbox>
						</el-checkbox-group>
						<div class="buttonGroup">
							<el-button size="mini" type="primary" @click="checkPopoverSubmit(item)"><%=rb.getString("QueDing")%></el-button>
							<el-button size="mini" @click="checkPopoverHide(item,'del')"><%=rb.getString("QuXiao")%></el-button>
						</div>	
					</div>
					<div slot="reference" class="ItemAndIconBoxCls">
						<i class="el-icon el-icon-filterAdd"></i>
						{{item.label}}
					</div>
				</el-popover>
			</div>
		</div>
	</div>
	<div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
		<%=rb.getString("QingKongShaiXuan")%>
	</div>
	<div class="advancedQueryItemBox" style="background: #FFF;position:absolute;right:10px;top:7px;" @click="timeLockChange">
		<i :class="{'el-icon':true, 'placeholder-bt':true, 'placeholder-bt':true, 'last-bt':true, 'el-icon-monitor-lock':timeLocked, 'el-icon-monitor-refresh':!timeLocked}" 
			style="cursor:pointer;position:relative;margin-right:5px;"></i>
		<span v-show="timeLocked"><%=rb.getString("SuoDingShuaXin")%></span>
		<span v-show="!timeLocked"><%=rb.getString("DingShiShuaXin")%></span>
	</div>
	
</div>
<el-dialog :visible.sync="collectInfoShow" top="10vh">
	<div slot="title">
		<span class="el-dialog__title"><%=rb.getString("XinXi")%></span>
		<i class="el-icon el-icon-operation-export" style="position: absolute;right: 45px;top: 9px;" @click="downloadMsg"></i>
	</div>
	<el-input v-model="collectContent" type="textarea" rows="25" readonly class="none-border"></el-input>
</el-dialog>

<script>
	var queryVue = new Vue({
			el:'#toolbar_tableHomeCellList',
			data(){
				return{
					placeholderText:'<%=rb.getString("QingShuRu")%>',
					advancedQueryItemList:[
						{
							type:'checkbox',
							isShow:true,
							isIndeterminate:false,
							checkAll:false,
							popoverShow:false,
							checkedItemList:[],
							oldCheckedItemList:[],
							label:'<%=rb.getString("ZaiXianZhuangTai") %>',
							options:[
								{label:'<%=rb.getString("LianJieZhengChang")%>',value:"1"},
								{label:'<%=rb.getString("LianJieDuanKai")%>',value:"0"},
								{label:'<%=rb.getString("TongBuZhong")%>',value:"3"},
								{label:'<%=rb.getString("TongBuShiBai")%>',value:"2"},
								{label:'<%=rb.getString("ChuShiHuaZhong")%>',value:"4"},
								{label:'<%=rb.getString("YuanTongBuZhong")%>',value:"5"},
								{label:'<%=rb.getString("YuanTongBuWanCheng")%>',value:"6"}
							],
							value:'connection_status',
						},
						{
							type:'select',
							isShow:true,
							popoverShow:false,
							selectVal:'',
							label:'<%=rb.getString("ShiFouJiHuo") %>',
							options:[
								{label:'<%=rb.getString("QuanBu")%>',value:''},
								{label:'<%=rb.getString("JiHuo")%>',value:"1"},
								{label:'<%=rb.getString("QuJiHuo")%>',value:"0"}
							],
							value:'op_state',
						},
						{
							type:'checkbox',
							isShow:true,
							isIndeterminate:false,
							checkAll:false,
							popoverShow:false,
							checkedItemList:[],
							oldCheckedItemList:[],
							label:'<%=rb.getString("ChanPinLeiXingBiaoZhi") %>',
							options:[],
							value:'product_model',
						},
						{
							type:'checkbox',
							isShow:false,
							isIndeterminate:false,
							checkAll:false,
							popoverShow:false,
							checkedItemList:[],
							oldCheckedItemList:[],
							label:'<%=rb.getString("SheBeiXingHaoMing") %>',
							options:[],
							value:'model_name',
						},
						{
							type:'checkbox',
							isShow:false,
							isIndeterminate:false,
							checkAll:false,
							popoverShow:false,
							checkedItemList:[],
							oldCheckedItemList:[],
							label:'<%=rb.getString("SoftwareVersion") %>',
							options:[],
							value:'software_version',
						},
						{
							type:'checkbox',
							isShow:false,
							isIndeterminate:false,
							checkAll:false,
							popoverShow:false,
							checkedItemList:[],
							oldCheckedItemList:[],
							label:'<%=rb.getString("FirmwareVersion") %>',
							options:[],
							value:'firmware_version',
						},
						{
							type:'checkbox',
							isShow:false,
							isIndeterminate:false,
							checkAll:false,
							popoverShow:false,
							checkedItemList:[],
							oldCheckedItemList:[],
							label:'<%=rb.getString("SheBeiZu") %>',
							options:[],
							value:'group_id',
						},
						{
							type:'select',
							isShow:false,
							popoverShow:false,
							selectVal:'',
							label:'<%=rb.getString("HaloBKaiGuan") %>',
							options:[
								{label:'<%=rb.getString("QuanBu")%>',value:''},
								{label:'<%=rb.getString("Kai")%>',value:"1"},
								{label:'<%=rb.getString("Guan")%>',value:"0"}
							],
							value:'halob_flag',
						},
						{
							type:'filter',
							popoverShow:false,
							checkedItemList:['connection_status','op_state','product_model'],
							oldCheckedItemList:['connection_status','op_state','product_model'],
							label:'<%=rb.getString("TianJiaShuaiXuan") %>',
							options:[
								{label:'<%=rb.getString("ZaiXianZhuangTai") %>',value:"connection_status"},
								{label:'<%=rb.getString("ShiFouJiHuo")%>',value:"op_state"},
								{label:'<%=rb.getString("ChanPinLeiXingBiaoZhi")%>',value:"product_model"},
								{label:'<%=rb.getString("SheBeiXingHaoMing")%>',value:"model_name"},
								{label:'<%=rb.getString("SoftwareVersion")%>',value:"software_version"},
								{label:'<%=rb.getString("FirmwareVersion")%>',value:"firmware_version"},
								{label:'<%=rb.getString("SheBeiZu")%>',value:"group_id"},
								{label:'<%=rb.getString("HaloBKaiGuan")%>',value:"halob_flag"},
							],
							value:'add_filter',
						},
					],

					isExisted: false,
					collectDeviceCode: '',
					collectSn: '',
					taskTime: '',
					msgExtend: false,
					collectInfoShow: false,
					collectContent: '',

					search_text:'',
					
					queryParams:{
						TimeZone : timeZone
					},

					timeLocked: true,
					enbAdditionalColShow:enbAdditionalColShow,
					northOperatorScenario:northOperatorScenario,
					percentage: ''
				}
			},
			computed: {
				isAdmin() {
					return is_super_user == 'true'
				},
				progressHide() {
					return ['',null,undefined].includes(this.percentage);
				}
			},
			methods:{
				queryProgress() {
					var vm = this,
						url = '${ctx}/cell/cpeinfos/getExportENBCellsProgress.action';

					clearInterval(progressInterval);
					progressInterval = setInterval(function() {
						axios.post(url).then(function(res){
							var data = res.data;

							vm.percentage = data - 0;

							if(vm.percentage == 100 || data === '') {
								clearInterval(progressInterval);
								setTimeout(function(){
									vm.percentage = '';
								},1500)
							}
						});
					},1000);
				},
				timeLockChange() {
					var vm = this;
            		
					vm.timeLocked = !vm.timeLocked;
				},
				// 模糊搜索
				query(){
					var vm = this;
					vm.queryParams.search_text = this.search_text;
					if(vm.enbAdditionalColShow == 'true' && vm.northOperatorScenario == 'S0009'){
						vm.queryParams['like_fields'] = 'serial_number,host_name,cell_ip,mac_address,cell_identity,phycellid,sub_station_name';
					}else{
						vm.queryParams['like_fields'] = 'serial_number,host_name,cell_ip,mac_address,cell_identity,phycellid';
					}
					Object.assign(enbvm.queryParams, vm.queryParams);
				},
				// 搜索域聚焦事件
				queryInputFocus(){
					var vm = this;
					if(vm.enbAdditionalColShow == 'true' && vm.northOperatorScenario == 'S0009'){
						vm.placeholderText = '<%=rb.getString("XiaoZhanBianMa")%>/ <%=rb.getString("HostName")%>/ IP/ MAC/ ECI/ PCI / <%=rb.getString("ZhanZhiMingCheng")%>';
					}else{
						vm.placeholderText = '<%=rb.getString("XiaoZhanBianMa")%>/ <%=rb.getString("HostName")%>/ IP/ MAC/ ECI/ PCI';
					}
				},
				// 搜索域失焦事件
				queryInputBlur(){
					var vm = this;
					vm.placeholderText = '<%=rb.getString("QingShuRu")%>';
				},
				// 高级查询 确定事件
				advanceQuery(type,paramsItem,value){
					var vm = this,
						params ={};
					if(type == 'select'){
						params[paramsItem] = value;
					}else{
						params[paramsItem] = value.join(',');
					}
					Object.assign(enbvm.queryParams, params);
				},
				// 高级搜索下拉全选事件
				handleCheckedAllChange(item){
					var vm = this,
					    allList = [];
					item.options.map((items)=>{
						if(items.value){
							allList.push(items.value)
						}
					})
					vm.advancedQueryItemList.map((items)=>{
						if(item.value == items.value){
							 items.checkedItemList = items.checkAll? allList : [];
							 items.isIndeterminate = false;
						}
					})
				},
				// 高级搜索下拉单选事件
				handleCheckedChange(item){
					var vm = this,
						checkCount = item.checkedItemList.length,
						allCount = item.options.length;
					
					vm.advancedQueryItemList.map((items)=>{
						if(item.value == items.value){
							if(items.type == 'checkbox'){
								items.checkAll = checkCount === allCount;
								items.isIndeterminate = checkCount > 0 && checkCount < allCount;
							}
						}
					})
				},
				// 高级搜索下拉弹窗 展开事件
				checkPopoverShow(item){
					var vm = this;
					vm.advancedQueryItemList.map((items)=>{
						if(item.value == items.value){
							items.popoverShow = true;
						}
					})
				},
				// 高级搜索下拉弹窗 收起事件
				checkPopoverHide(item,types){
					var vm = this,
						checkCount = 0,
						allCount = item.options.length;
					if(item.type != 'select'){
						checkCount = item.oldCheckedItemList.length
					}
					vm.advancedQueryItemList.map((items)=>{
						if(item.value == items.value){
							items.popoverShow = false;
							if(items.type != 'select'){
								items.checkedItemList = items.oldCheckedItemList;
							}
							if(items.type == 'checkbox'){
								items.checkAll = checkCount === allCount;
								items.isIndeterminate = checkCount > 0 && checkCount < allCount;
							}
						}
					})
					if(types == 'del'){
						document.body.click();
					}
				},
				// 高级搜索下拉弹窗 选择提交
				checkPopoverSubmit(item){
					var vm = this,
						params ={};
					if(item.type == 'checkbox'){
						vm.advancedQueryItemList.map((items)=>{
							if(item.value == items.value){
								items.oldCheckedItemList = items.checkedItemList;
								if(items.checkAll){
									params[items.value] = '';
								}else{
									params[items.value] = items.checkedItemList.join(',');
								}
							}
						})
					}else{
						vm.advancedQueryItemList.map((items)=>{
							if(item.value == items.value){
								items.oldCheckedItemList = items.checkedItemList;
							}else{
								var result = item.checkedItemList.includes(items.value);
								if(!result){
									items.isShow = false;
									items.checkedItemList = [];
									items.oldCheckedItemList = [];
									items.checkAll = false;
									items.isIndeterminate = false;
									params[items.value] = '';
								}else{
									items.isShow = true;
								}
							}
						})
					}
					
					Object.assign(enbvm.queryParams, params);
					document.body.click();
				},
				// 筛选项删除事件
				checkItemDel(item){
					var vm = this,
						params ={};
					vm.advancedQueryItemList.map((items)=>{
						if(item.value == items.value){
							items.isShow = false;
							if(items.type != 'select'){
								items.checkedItemList = [];
								items.oldCheckedItemList = [];
								items.checkAll = false;
								items.isIndeterminate = false;
							}else{
								items.selectVal = '';
							}
							params[items.value] = '';
						}
						if(items.value == 'add_filter'){
							var idx = items.checkedItemList.indexOf(item.value),
								oldIdx = items.oldCheckedItemList.indexOf(item.value);
							if(idx>-1){
								items.checkedItemList.splice(idx,1)
							}
							if(oldIdx>-1){
								items.oldCheckedItemList.splice(idx,1)
							}
						}

					})
					Object.assign(enbvm.queryParams, params);
				},
				//单选 选择事件
				selectChangeClick(value,item){
					var vm = this,
						params ={};
					
					vm.advancedQueryItemList.map((items)=>{
						if(item.value == items.value){
							items.selectVal = value;
							params[items.value] = value;
						}
					})
					Object.assign(enbvm.queryParams, params);
					document.body.click();
				},
				// 清除筛选
				clearFilterClick(){
					var vm = this,
						params = {
							halodSerialNumbers:''
						};
					vm.advancedQueryItemList.map((items)=>{
						if(items.isShow && items.isShow== true ){
							if(items.type == 'checkbox'){
								items.checkedItemList = [];
								items.oldCheckedItemList = [];
								items.checkAll = false;
								items.isIndeterminate = false;
							}else if(items.type == 'select'){
								items.selectVal = '';
							}
							if(items.type == 'checkbox' || items.type == 'select'){
								params[items.value] = '';
							}
						}
					})
					Object.assign(enbvm.queryParams, params);
					document.body.click();
				},
				queryLatestInfo() {
					var vm = this,
                        params = {
                            type: 'enb',
                            operatorCode: operatorCodeGloab
                        };

                    axios.post('${ctx}/trace/queryLatestMessageTraceDeviceInfo.action', stringify(params)).then(function(res){
                        var data = res.data;

						if(data) {
                        	vm.collectSn = data.serialNumber;
							vm.collectDeviceCode = data.deviceCode;

							if(data.status == '0' && data.remainTime) {
								vm.startInterval(data.remainTime);
							}else if(data.status == '1'){
								vm.taskTime = '';
								vm.startInterval('00:01');
							}

							if(['',undefined].includes(data.status) && ['',undefined].includes(data.serialNumber)) {
								vm.isExisted = false;
							}else {
								vm.isExisted = true;
							}
						}
                    });
				},
                viewMsg() {
					var vm = this,
                        params = {
                            serialNumber: vm.collectSn,
                            type: 'enb'
                        };
                           
                    vm.collectInfoShow = true;

                    axios.post('${ctx}/trace/queryMessageTraceInfo.action', stringify(params)).then(function(res){
                        var data = res.data;

                        if(data) {
							vm.collectContent = data;
                    		vm.collectInfoShow = true;
                        }
                    });
                },
                downloadMsg() {
                    var vm = this,
                        params = {
                            serialNumber: vm.collectSn,
                            type: 'enb',
                            timeZone: timeZone
                        };

                    exportByForm('${ctx}/trace/downLoadMessageTraceInfo.action',params);
                },
				clearMsg() {
					var vm = this,
                        params = {
                            deviceCode: vm.collectDeviceCode,
                            serialNumber: vm.collectSn,
                            type: 'enb',
                            operatorCode: operatorCodeGloab
                        };

                    axios.post('${ctx}/trace/clear.action', stringify(params)).then(function(res){
                        var data = res.data;

                        if(data.success == true) {
                            vm.$message.success('<%=rb.getString("ChengGong")%>');
							vm.queryLatestInfo();
                        }else {
                            vm.$message.error(data.message);
                        }
                    });
				},
				stopCollect() {
					var vm = this,
                        row = enbvm.selectedRow || {},
                        params = {
                            deviceCode: vm.collectDeviceCode,
                            serialNumber: vm.collectSn,
                            type: 'enb',
                            operatorCode: operatorCodeGloab
                        };

                    axios.post('${ctx}/trace/stop.action', stringify(params)).then(function(res){
                        var data = res.data;

                        if(data.success == true) {
                            vm.$message.success('<%=rb.getString("ChengGong")%>');
							vm.queryLatestInfo();
                        }else {
                            vm.$message.error(data.message);
                        }
                    });
				},
				startInterval(time) {
					var vm = this,
						list = time.split(':'),
						totalTime = list[0]*60 + list[1]*1;

					clearInterval(collectInterval);
					
					collectInterval = setInterval(function() {
						totalTime -= 1;
						vm.taskTime = vm.formaterTime(Math.floor(totalTime/60)) + ':' + vm.formaterTime(totalTime%60);

						if(totalTime<1) {
							clearInterval(collectInterval);
							vm.taskTime = '';
						}
					},1000);
				},
				formaterTime(num) {

					return num < 10? '0'+num : num;
				},
				// 初始化
				init(){
					var vm = this;
					axios.post("${ctx}/cell/cpeinfos/getCellVersionList.action").then(function(response){
						var data = response.data ? response.data : [];
						var arr = [];
						data.map(function(item){
							arr.push({label:item.software_version,value:item.software_version})
						})
						vm.advancedQueryItemList.map((items)=>{
							if('software_version' == items.value){
								items.options = arr
							}
						})
					})
					axios.post("${ctx}/cell/cpeinfos/getFirmwareVersionList.action?isGnb=0").then(function(response){
						var data = response.data ? response.data : [];
						var arr = [];
						data.map(function(item){
							arr.push({label:item.firmware_version,value:item.firmware_version})
						})
						vm.advancedQueryItemList.map((items)=>{
							if('firmware_version' == items.value){
								items.options = arr
							}
						})
					})
					axios.post("${ctx}/cell/cpeinfos/getModelNameList.action?isGnb=0").then(function(response){
						var data = response.data ? response.data : [];
						var arr = [];
						data.map(function(item){
							arr.push({label:item.module_type,value:item.module_type})
						})
						vm.advancedQueryItemList.map((items)=>{
							if('model_name' == items.value){
								items.options = arr
							}
						})
					})
					axios.post("${ctx}/cell/cpeinfos/getEnbMonitorProductList.action?isGnb=0").then(function(response){
						var data = response.data ? response.data : [];
						var arr = [];
						data.map(function(item){
							if (item){
								arr.push({label:item,value:item})
							}
						})
						vm.advancedQueryItemList.map((items)=>{
							if('product_model' == items.value){
								items.options = arr
							}
						})
					})
					axios.post("${ctx}/cell/cpeinfos/getDeviceGroupListByCell.action").then(function(response){
						var data = response.data ? response.data : [];
						var arr = [];
						data.map(function(item){
							if (item){
								arr.push({label:item.group_name,value:item.id})
							}
						})
						vm.advancedQueryItemList.map((items)=>{
							if('group_id' == items.value){
								items.options = arr;
							}
						})
					})
				},
				columnSetting() {
					$("#showOrHideItem").slideDown();
				},
				reseteNBQuery(){
					var vm = this;
					vm.clearFilterClick();
					enbvm.queryParams.halodSerialNumbers = '';
				},
			},
			mounted(){
				this.init();
				this.queryLatestInfo();

				setTimeout(closeLoading, 500);
                //closeLoading();
			}
		})
</script>
