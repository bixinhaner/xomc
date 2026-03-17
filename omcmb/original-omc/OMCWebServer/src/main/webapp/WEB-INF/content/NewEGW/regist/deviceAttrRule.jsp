<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
    #deviceAttrRulePage .newIconBoxCls-bt .el-icon-close::before{
        font-size: 14px;
    }
    #deviceAttrRulePage .ruleListTableCls .ruleListTableClass .el-table__body tr:hover{
        cursor: move;
    }
    #deviceAttrRulePage .nameContainsContentBoxCls{
        display: flex;
        align-items: center;
        margin-bottom: 10px;
    }
    #deviceAttrRulePage .nameContainsContentBoxCls:last-child{
        margin-bottom: 0px;
    }
    #deviceAttrRulePage .nameContainsContentBoxCls .el-select{
        width: 120px;
    }
    #deviceAttrRulePage .nameContainsContentBoxCls .el-input-group__prepend div.el-select .el-input__inner{
        border: 0;
    }
    #deviceAttrRulePage .nowInputNameContainsContentBoxCls{
        width: 100%;
        font-size: 12px;
        color: rgba(0, 0, 0, 0.32);
        margin-bottom: 20px;
        word-wrap: break-word;
        word-break: break-all;
        white-space: normal;
        line-height: 1.5;
    }
    /* 输入框错误状态样式 */
    #deviceAttrRulePage .nameContainsContentBoxCls .input-error .el-input__inner{
        border-color: #F56C6C !important;
    }
    #deviceAttrRulePage .nameContainsContentBoxCls .input-error.is-focus .el-input__inner{
        border-color: #F56C6C !important;
    }
    /* 右侧弹窗宽度 */
    #deviceAttrRulePage .tablesListRightBoxCls {
        flex: 0 1 500px !important;
    }
</style>
<!-- 设备归属设备组规则页面 -->
<div class="pageDefault" id='deviceAttrRulePage'>
	<div class="devicesMainBoxCls">
        <!--设备归属设备组规则列表-->
        <div class="tablesListBoxCls">
            <div class="tablesListMainBoxCls">
                <div class="ruleListTableCls" style="position: relative;">
                    <span class="el-icon-drag el-icon" style="position: absolute;left: 15px; top: 65px;z-index: 99;"></span>
                    <el-ctable 
                        id="ruleListTable"
                        ref="ruleListTable" 
                        class="ruleListTableClass"
                        :data="ruleListTableData" 
                        :height="height" 
                        :row-key="'id'" 
                        :query-params="queryRuleParams"
                        :pagination="false"
                        :rownumber=true 
                    >
                        <template slot='toolbar' style="position: relative;">
                            <div style="display: flex;align-items: center;">
                                <span style="margin:0px 0px 4px 20px;color:#363B4E;font-size:14px;font-weight:bold;"><%=rb.getString("GuiZeBiaoTi")%></span>
                                <el-query type="normal" @query="queryRules" placeholder="<%=rb.getString("CaoZuo")%>"></el-query>	
                            </div>
                            <!-- 按钮   添加规则 -->
                            <div class="newIconBoxCls-bt" style="right: 60px;top: 15px" @click="openAddRule" tip="<%=rb.getString("TianJia")%>">
                                <span class="el-icon el-icon-plus"></span>
                            </div>
                            <!-- 按钮   关闭 -->
                            <div class="newIconBoxCls-bt" style="right:20px;top: 15px" @click="closeRulePage" tip="<%=rb.getString("GuanBi")%>">		
                                <span class="el-icon-close el-icon"></span>
                            </div>
                        </template>
                        <!--设备列表-->
                        <el-table-column width="110" prop="" class-name="no-text-tips">
                            <template slot-scope="scope">
                                <div style="padding-top: 5px;">
                                    <div v-if="scope.row.enable == '0'" class="el-icon el-icon-operation-start disabled"></div>
                                    <div v-if="scope.row.enable == '1'" class="el-icon el-icon-operation-start grayIcon" @click="revisitStartRule(scope.row,event)"></div>
                                    <div class="el-icon el-icon-operation-edit grayIcon" @click="editRule(scope.row,event)" style="margin-left: 10px;"></div>
                                    <div class="el-icon el-icon-operation-delete grayIcon" @click="deleteRule(scope.row,event)" style="margin-left: 10px;"></div>
                                </div>
                            </template>
                        </el-table-column>
                        <el-table-column key="enable" label='Enable' width="80" prop="enable">
                            <template slot-scope="scope">
                                <div class="switchBoxCls-ctn" @click="ruleEnableChange(scope.row, event)">
                                    <el-switch v-model="scope.row.enable" active-value="1" inactive-value="0" style="zoom: 0.8" ></el-switch>
                                </div>
                            </template>
                        </el-table-column>
                        <el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="320" prop="operators"></el-table-column>
                        <el-table-column label='<%=rb.getString("ChuangJianShiJian")%>' min-width="120" prop="create_time"></el-table-column>
                    </el-ctable>
                </div>
            </div>
            <div class="tablesListRightBoxCls" v-show="ruleListRightBoxShow">
                <div class="rightOutBoxHeadCls">
                    <span>{{ruleRightBoxTitle}}</span>
                    <span class="el-icon el-icon-close greyIcon" @click="ruleRightBoxClose"></span>
                </div>
                <div class="rightItemMainBox" v-if="ruleRightType == 'add' || ruleRightType == 'edit'">
                    <el-form  :model="addRuleForm" ref="addRuleForm" :rules="addRuleFormRules" label-position="top" id="addRuleForm">
                        <el-form-item label='<%=rb.getString("MuBiaoSheBeiZu")%>' prop='move_to_group_id' style='margin-bottom: 20;'>
                            <el-select v-model="addRuleForm.move_to_group_id" filterable>
                                <el-option v-for="item in deviceGroupOptions" :key="item.id" :label="item.group_name" :value="item.id"></el-option>
                            </el-select>
                        </el-form-item>
                        <el-form-item label="Enable" prop="enable" style='margin-bottom: 20px;'>
                            <el-switch v-model="addRuleForm.enable" active-value="1" inactive-value="0"></el-switch>
                        </el-form-item>
                        <el-form-item prop="matching_mode" style='margin-bottom: 20px;'>
                            <span slot="label">
                                <%=rb.getString("PiPeiGuiZe")%>
                                <span v-if="addRuleForm.matching_mode == 'deviceName'" style="color: rgba(0,0,0,0.32);">（<%=rb.getString("BuChaoGuo")%> 10）</span>
                            </span>
                            <el-radio-group v-model="addRuleForm.matching_mode" @change="matchingModeChange">
                                <el-radio label="deviceName" border size="small"><%=rb.getString("Title_SheBeiMingCheng")%></el-radio>
                                <el-radio v-if="isSupportGSM && deviceType == 'ENB'" label="lac" border size="small">LAC</el-radio>
                                <el-radio v-if="deviceType !== 'CPE'" label="tac" border size="small">TAC</el-radio>
                            </el-radio-group>
                        </el-form-item>
                        <el-form-item v-if="addRuleForm.matching_mode == 'deviceName'" style='margin-bottom: 0px;'>
                            <div v-for="(filter, index) in nameContainsContentList" :key="index" class="nameContainsContentBoxCls">
                                <!-- 第一行：只有 filterCondition 下拉和输入框 -->
                                <div v-if="index === 0" style="display: flex; align-items: center; gap: 5px;">
                                    <el-select v-model="filter.condition" style="width: 120px;">
                                        <el-option 
                                            v-for="option in filterConditionOptions" 
                                            :key="option.value"
                                            :label="option.label" 
                                            :value="option.value">
                                        </el-option>
                                    </el-select>
                                    <el-input 
                                        v-model="filter.value" 
                                        style="width: 325px;"
                                        :class="{'input-error': filter.hasError}"
                                        maxlength="64"
                                        @blur="validateSingleInput(index)"
                                        @input="validateSingleInput(index)">
                                    </el-input>
                                </div>
                                <!-- 其他行：And/Or 下拉、filterCondition 下拉和输入框 -->
                                <div v-else style="display: flex; align-items: center; gap: 5px;">
                                    <el-select v-model="filter.andOr" style="width: 70px;">
                                        <el-option 
                                            v-for="option in getAndOrOptions(index)" 
                                            :key="option.value"
                                            :label="option.label" 
                                            :value="option.value"
                                            :disabled="option.disabled">
                                        </el-option>
                                    </el-select>
                                    <el-select v-model="filter.condition" style="width: 120px;">
                                        <el-option 
                                            v-for="option in filterConditionOptions" 
                                            :key="option.value"
                                            :label="option.label" 
                                            :value="option.value">
                                        </el-option>
                                    </el-select>
                                    <el-input 
                                        v-model="filter.value" 
                                        style="width: 225px;"
                                        :class="{'input-error': filter.hasError}"
                                        maxlength="64"
                                        @blur="validateSingleInput(index)"
                                        @input="validateSingleInput(index)">
                                    </el-input>
                                    <span @click="removeFilter(index)" style="cursor: pointer;" class="el-icon el-icon-circle-close"></span>
                                </div>
                            </div>
                            <!-- 新增按钮放在列表下方 -->
                            <div @click="addNameContainsContent" v-if="nameContainsContentList.length < 10" class="addNameContainsContentBoxCls" style="margin-top: 10px;">
                                <span class="el-icon el-icon-plus" style="margin-right: 5px;"></span>
                                <span><%=rb.getString("TianJiaTiaoJian")%></span>
                            </div>
                        </el-form-item>
                        <el-form-item v-if="addRuleForm.matching_mode == 'deviceName'" prop="name_contains" style='margin-bottom: 20px;'>
                            <el-input v-model="addRuleForm.name_contains" v-show="false"></el-input>
                        </el-form-item>
                        <div v-if="nowInputNameContainsContent" class="nowInputNameContainsContentBoxCls">{{nowInputNameContainsContent}}</div>
                        <el-form-item v-if="addRuleForm.matching_mode == 'tac' || addRuleForm.matching_mode == 'lac'" :label="addRuleForm.matching_mode == 'lac' ? 'LAC': 'TAC'" prop="tac_rag" style='margin-bottom: 20px;'>
                            <el-input v-model="addRuleForm.tac_rag" maxlength="50" placeholder="eg: 1,2,3,1-3"></el-input>
                        </el-form-item>
                    </el-form>
                </div>
                <div class="rightItemMainBox" v-if="ruleRightType == 'active'">
                    <div style="margin-bottom: 10px;font-size: 14px;"><%=rb.getString("ZhiXingGuiZeDeSheBeiZu")%></div>
                    <div style="height: calc(100% - 40px); padding-top: 5px;border: 1px solid #D5DCEC;">
                        <el-tree 
                            ref="groupTree"
                            :data="groupData"
                            node-key="id"
                            show-checkbox
                            :default-expanded-keys="defaultexpandedKeys"
                            :default-checked-keys="defaultCheckedKeys"
                            :props= "{label:'group_name'}"
                            :highlight-current="true"
                        >
                            <div class="treeItemBoxCls" slot-scope="{ node,data }">
                                <span>{{node.label}}</span>
                            </div>
                        </el-tree>
                    </div>
                </div>
                <div class="footer">
                    <div class="lnkbuttonGroup" style="margin-left:20px;" v-if="ruleRightType == 'add' || ruleRightType == 'edit'">
                        <el-button  type="primary"  @click="addRuleSubmit"><%=rb.getString("QueDing")%></el-button>
                        <el-button  @click="ruleRightBoxClose"><%=rb.getString("QuXiao")%></el-button>
                    </div>
                    <div class="lnkbuttonGroup" style="margin-left:20px;" v-if="ruleRightType == 'active'">
                        <el-button  type="primary"  @click="activeRuleSubmit"><%=rb.getString("QueDing")%></el-button>
                        <el-button  @click="ruleRightBoxClose"><%=rb.getString("QuXiao")%></el-button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>
<script type="text/javascript">
var deviceAttrRuleVue = new Vue({
	el:'#deviceAttrRulePage',
	data(){
        var vm = this;
        var validTAC = function(rule,value,callback) {
                var message = vm.deviceType == 'ENB' ? 'eg: 1,2,3,1-3;Range, 0-65535' : 'eg: 1,2,3,1-3;Range, 0-16777215';
                if(vm.addRuleForm.matching_mode == 'tac' || vm.addRuleForm.matching_mode == 'lac'){
                    if(value == '' || value == null || value == undefined){
                        callback(message);
                    }else{
                        if(vm.deviceType == 'ENB'){
                            if(!vm.isValidIntegerTacRange(value,0,65535)){
                                callback(message);
                            }else{
                                callback();
                            }
                        }else{
                            if(!vm.isValidIntegerTacRange(value,0,16777215)){
                                callback(message);
                            }else{
                                callback();
                            }
                        }
                    }
                }else{
                    callback();
                }
            },
            validNameContainsContent = function(rule,value,callback) {

                if(vm.addRuleForm.enable == '1' && vm.addRuleForm.matching_mode == 'deviceName'){
                    var hasError = false;
                    
                    // 遍历所有输入框，检查是否有空值
                    vm.nameContainsContentList.forEach(function(filter, index){
                        var isEmpty = !filter.value || filter.value.trim() === '';
                        if(isEmpty){
                            vm.$set(filter, 'hasError', true);
                            hasError = true;
                        } else {
                            vm.$set(filter, 'hasError', false);
                        }
                    });
                    
                    if(hasError){
                        callback(new Error('Please input required field'));
                        return;
                    }
                    
                    callback();
                }else{
                    // 清除所有错误状态
                    vm.nameContainsContentList.forEach(function(filter){
                        vm.$set(filter, 'hasError', false);
                    });
                    callback();
                }
            };
		return {
            rowData: '',
            height:'100%',
            deviceGroupOptions:[],
            ruleListTableData:[],
            queryRuleParams: {
                searchText: '',
            },
            ruleListRightBoxShow:false,
            ruleRightType:'',
            ruleRightBoxTitle: '',
            addRuleForm: {
                move_to_group_id: '',
                enable: '0',
                matching_mode: 'deviceName',
                name_contains: '',
                tac_rag: ''
            },
            addRuleFormRules: {
                name_contains:[{validator:validNameContainsContent}],
                tac_rag:[{validator:validTAC}],
            },
            groupData:[],
            defaultexpandedKeys:[],
			defaultCheckedKeys:[],
            isSupportGSM: supportGSM,
            
            // 过滤器选项配置
			filterConditionOptions: [
                {label: '<%=rb.getString("BaoHan")%>', value: 'contain', minIndex: 0},
				{label: '<%=rb.getString("BuBaoHan")%>', value: 'notContain', minIndex: 0},
				{label: '<%=rb.getString("YiShenMoKaiShi")%>', value: 'startWith', minIndex: 0},
				{label: '<%=rb.getString("YiShenMoJieShu")%>', value: 'endWith', minIndex: 0}
			],
            // 或 与 下拉选项配置
            conditionAndOrOptions: [
                {label: '<%=rb.getString("Yu")%>', value: 'and'},
                {label: '<%=rb.getString("Huo")%>', value: 'or'}
			],
			// 过滤器列表
			nameContainsContentList: [
				{
					condition: 'contain',
					value: '',
					hasError: false
				}
			]
		}
	},
    computed:{
        nowInputNameContainsContent() {
			var vm = this,
				orGroups = [[]]; // 按 or 分组，初始有一个空组
			
			// 按 or 分组
			vm.nameContainsContentList.forEach(function(filter, index) {
				if (filter.value && filter.value.trim() !== '') {
					// 如果当前行是 or，创建新组
					if (index > 0 && filter.andOr === 'or') {
						orGroups.push([]);
					}
					
					// 将当前过滤器添加到最后一个组
					orGroups[orGroups.length - 1].push({
						condition: filter.condition,
						value: filter.value.trim()
					});
				}
			});
			
			// 过滤掉空组
			orGroups = orGroups.filter(function(group) {
				return group.length > 0;
			});
			
			if (orGroups.length === 0) {
				return '';
			}
			
			// 生成最终字符串
			var groupParts = [];
			orGroups.forEach(function(group) {
				var conditionParts = [];
				
				group.forEach(function(item) {
					// 查找 condition 对应的动词
					var verb = '';
					if (item.condition === 'contain') {
						verb = 'contain';
					} else if (item.condition === 'notContain') {
						verb = 'not contain';
					} else if (item.condition === 'startWith') {
						verb = 'start with';
					} else if (item.condition === 'endWith') {
						verb = 'end with';
					}
					
					conditionParts.push(verb + ' "' + item.value + '"');
				});
				
				// 组内用 and 连接
				var groupText = conditionParts.join(' and ');
				
				// 如果有多个 or 组，每个组都加括号；如果只有一个组但有多个条件，也加括号
				if (orGroups.length > 1 || group.length > 1) {
					groupText = '(' + groupText + ')';
				}
				
				groupParts.push(groupText);
			});
			
			// 组间用 or 连接
			var result = groupParts.join(' or ');
			
			// 如果有多个组或第一组有多个条件，添加 Must 前缀
			if (orGroups.length > 1 || orGroups[0].length > 1) {
				result = 'Must ' + result;
			} else {
				// 单个条件，首字母大写
				result = result.charAt(0).toUpperCase() + result.slice(1);
			}
			
			return result;
		},
        deviceType() {
            var codes={
                'eNB': 'ENB',
                'gNB': 'GNB',
                'CPE': 'CPE'
            }
			return codes[egwRegisterVue.deviceType];
		}
    },
	watch:{},
	methods:{
		// 初始化
		init(){
			var vm = this;
            vm.initSortable();
            vm.getRuleList();
            vm.queryGroupList();
            vm.queryDeviceGroupOption();
		},
        initSortable(){
            var vm = this;
            const tbody = document.querySelectorAll('.ruleListTableCls .el-table__body-wrapper tbody')[0];
            const sortable = new Sortable(tbody, {
                onEnd: function (evt) {
                    const curRow = vm.$refs.ruleListTable.rows.splice(evt.oldIndex, 1)[0];
                    vm.$refs.ruleListTable.rows.splice(evt.newIndex, 0, curRow);
                    let paramsList = [];
                    vm.$refs.ruleListTable.rows.map((item,index)=>{
                        paramsList.push({
                            id: item.id,
                            order: index + 1
                        })
                    })
                    let paramsData = JSON.stringify(paramsList);

                    axios.post('${ctx}/moveDeviceGroupRule/rule/order/update.action',paramsData,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
                       var data = response.data;
                    })
                }
            });
        },
        getRuleList(){
            var vm = this,
                urls = '${ctx}/moveDeviceGroupRule/rule/list/get.action',
                params = {
                    timeZone: timeZone,
                    searchText: vm.queryRuleParams.searchText,
                    deviceType: vm.deviceType
                };
            axios.post(urls,stringify(params)).then(function(response){
                var data = response.data;
                vm.ruleListTableData = data ? data : [];
            })
        },
        // 查询设备组下拉数据
        queryDeviceGroupOption(){
			var vm = this;
			axios.post('${ctx}/system/deviceGroup/getSimpleDeviceGroupList.action',stringify({isAll:'0'})).then(function(response){
				let data = response.data
				vm.deviceGroupOptions = data;
                if(vm.deviceGroupOptions.length>0){
					vm.addRuleForm.move_to_group_id = vm.deviceGroupOptions[0].id;
				}
			}).catch(function(error){});
		},
		// 设备组树结构数据查询
		queryGroupList(val){
			var vm =this,
				typeCode={
					'ENB': 0,
					'GNB': 1,
					'CPE': 2,
				},
				params={
					search_text:'',
					type: typeCode[vm.deviceType]
				};
			vm.defaultexpandedKeys =[];
			vm.defaultCheckedKeys = [];
			axios.post('${ctx}/system/deviceGroup/getFullDeviceGroupList.action',stringify(params)).then(function(response){
				let data = response.data
				if(data.rows.length>0){
					vm.groupData = data.rows;
					vm.defaultexpandedKeys.push(vm.groupData[0].id);
					vm.defaultCheckedKeys.push(vm.groupData[0].children[0].id);
				}
			}).catch(function(error){})
		},
		// 打开新增规则页面
        openAddRule(){
            var vm = this;
            vm.ruleRightType = 'add';
            vm.ruleRightBoxTitle = '<%=rb.getString("TianJia")%>';
            vm.ruleListRightBoxShow = true;
        },
        ruleRightBoxClose(){
            var vm = this,
                params = {
                    move_to_group_id: vm.deviceGroupOptions[0].id,
                    enable: '0',
                    matching_mode: 'deviceName',
                    name_contains: '',
                    tac_rag: ''
                };
            Object.assign(vm.addRuleForm,params);
            
            // 重置 nameContainsContentList 为默认状态
            vm.nameContainsContentList = [{
                condition: 'contain',
                value: '',
                hasError: false
            }];
            
            vm.ruleListRightBoxShow = false;
        },
        // 规则列表查询
        queryRules(val){
            var vm = this;
            vm.queryRuleParams.searchText = val;
            vm.getRuleList();
        },
        // 规则开关改变事件
        ruleEnableChange(row){
            var vm = this,
				urls = '${ctx}/moveDeviceGroupRule/rule/enable/update.action',
				params = {
                    id: row.id,
					enable:row.enable == '1'? '0' : '1'
				};
			axios.post(urls,stringify(params)).then(function(response){
				var data = response.data;
				var message = '<%=rb.getString("ChengGong")%>';
				if(data["success"]){
					vm.$message({
						message:message,
						type:'success',
					})
                    vm.getRuleList();
				}else{
					vm.$message.error(data["message"])
				}
			})
			event.stopPropagation();
        },
        // 重新执行规则
        revisitStartRule(row){
            var vm = this;
            vm.rowData = row;
            vm.ruleRightType = 'active';
            vm.ruleRightBoxTitle = 'Active';
            vm.ruleListRightBoxShow = true;        
        },
        activeRuleSubmit(){
            var vm = this,
                urls = '${ctx}/moveDeviceGroupRule/rule/execute.action',
                groupIdList = [],
                params = {
                    id:vm.rowData.id,
                    groupIds:''
                };
            let activeGroupList = vm.$refs.groupTree.getCheckedNodes();
            if(activeGroupList.length === 0){
                vm.$message({
                    message:'<%=rb.getString("QingXuanZeSheBeiZu")%>',
                    type:'warning',
                });
                return
            }
            activeGroupList.map((item)=>{
                if(!item.children){
                    groupIdList.push(item.id)
                }
            })
            params.groupIds = groupIdList.join(',');
            axios.post(urls,stringify(params)).then(function(response){
                var data = response.data;
                if(data["success"]){
                    vm.$message({
                        message:'<%=rb.getString("ChengGong")%>',
                        type:'success',
                    })
                    vm.ruleRightBoxClose();
                }else{
                    vm.$message.error(data["message"])
                }
            }) 
        },
        // 编辑规则
        editRule(row){
            var vm = this;
            vm.rowData = row;
            vm.ruleRightType = 'edit';
            vm.ruleRightBoxTitle = '<%=rb.getString("XiuGai")%>';
            Object.keys(vm.addRuleForm).forEach(key => {
                vm.addRuleForm[key] = row[key];
            });
            
            // 从 nameRuleList 回显到 nameContainsContentList
            if(row.nameRuleList && row.nameRuleList.length > 0){
                vm.nameContainsContentList = row.nameRuleList.map(function(item){
                    return {
                        condition: item.condition,
                        value: item.value,
                        andOr: item.andOr,
                        hasError: false
                    };
                });
            } else {
                // 如果没有 nameRuleList，添加一个默认的空过滤器
                vm.nameContainsContentList = [{
                    condition: 'contain',
                    value: '',
                    hasError: false
                }];
            }
            
            vm.ruleListRightBoxShow = true;
        },
        // 删除规则
        deleteRule(row){
            var vm = this,
                urls = '${ctx}/moveDeviceGroupRule/rule/del.action',
                params={
                    id:row.id
                };
            vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
                confirmButtonText:'<%=rb.getString("QueDing")%>',
                cancalButtonText:'<%=rb.getString("QuXiao")%>',
                type:'warning'
            }).then(()=>{
                axios.post(urls,stringify(params)).then(res=>{
                    var data = res.data;
                    if(data["success"]){
                        vm.$message({
                            message: '<%=rb.getString("ChengGong")%>',
                            type:'success',
                        });
                        vm.getRuleList();
                    }else{
                        vm.$message.error(data["message"])
                    }
                })
            }).catch(()=>{})
        },
        addRuleSubmit(){
            var vm = this,
                urls = '',
                params = {
                    device_type: vm.deviceType,
                    order: ''
                };
            Object.assign(params, vm.addRuleForm);
            
            // 使用 nameRuleList 传递过滤器数据
            params.nameRuleList = vm.nameContainsContentList.map(function(item){
                return {
                    condition: item.condition,
                    value: item.value,
                    andOr: item.andOr
                };
            });
            
            if(vm.ruleRightType == 'add'){
                params.order = vm.ruleListTableData.length + 1;
                urls = '${ctx}/moveDeviceGroupRule/rule/add.action';
            }else{
                params.id = vm.rowData.id;
                params.order = vm.rowData.order;
                urls = '${ctx}/moveDeviceGroupRule/rule/update.action';
            }
            let paramsData = JSON.stringify(params);
            vm.$refs.addRuleForm.validate((valid) => {
                if (valid) {
                    axios.post(urls,paramsData,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
                        let data = response.data;
                        if (data.success){
                            vm.$message({
                                message: '<%=rb.getString("ChengGong")%>',
                                type:'success',
                            })
                            vm.ruleRightBoxClose();
                            vm.getRuleList();//刷新列表
                        }else {
                            vm.$message.error(data.message)
                        }
                    }).catch(function(error){})
                } else {
                    return false;
                }
            });
        },
        closeRulePage(){
            var vm = this;
			egwRegisterVue.sharingSlideCancel();
        },
        matchingModeChange(){
            var vm = this;
            vm.addRuleForm.tac_rag = '';
            vm.addRuleForm.name_contains = '';
            vm.$refs.addRuleForm.clearValidate('tac_rag');
            vm.$refs.addRuleForm.clearValidate('name_contains');
            
            // 重置 nameContainsContentList 为默认状态
            vm.nameContainsContentList = [{
                condition: 'contain',
                value: '',
                hasError: false
            }];
        },
        // 添加过滤器
		addNameContainsContent(){
			var vm = this;
			if(vm.nameContainsContentList.length < 10){
				// 检查前面是否有选择了 Or 的行
				var hasOrCondition = vm.nameContainsContentList.some(function(filter, index){
					return index > 0 && filter.andOr === 'or';
				});
				
				// 如果已有 Or，新增行默认为 Or；否则默认为 And
				var defaultAndOr = hasOrCondition ? 'or' : 'and';
				
				vm.nameContainsContentList.push({
					andOr: defaultAndOr,
					condition: 'contain',
					value: '',
					hasError: false
				});
			}
		},
		// 动态获取 And/Or 选项（Or下面不能出现And，Or上面相邻的And能改成Or）
		getAndOrOptions(index){
			var vm = this;
			
			// 检查当前行之前是否有选择了 Or 的行
			var hasOrBefore = false;
			for(var i = 1; i < index; i++){
				if(vm.nameContainsContentList[i].andOr === 'or'){
					hasOrBefore = true;
					break;
				}
			}
			
			// 检查下一行（相邻）是否为 And
			var nextIsAnd = false;
			if(index + 1 < vm.nameContainsContentList.length){
				if(vm.nameContainsContentList[index + 1].andOr === 'and'){
					nextIsAnd = true;
				}
			}
			
			// 如果之前有 Or，则禁用 And 选项
			// 如果下一行是 And，则禁用 Or 选项（Or上面相邻的And能改成Or，但Or下面不能是And）
			return vm.conditionAndOrOptions.map(function(option){
				return {
					label: option.label,
					value: option.value,
					disabled: (hasOrBefore && option.value === 'and') || (nextIsAnd && option.value === 'or')
				};
			});
		},
		// 删除过滤器
		removeFilter(index){
			var vm = this;
			if(vm.nameContainsContentList.length > 1){
				vm.nameContainsContentList.splice(index, 1);
				
				// 如果删除的是第一行，需要移除新的第一行的 andOr 字段
				if(index === 0 && vm.nameContainsContentList.length > 0){
					vm.$delete(vm.nameContainsContentList[0], 'andOr');
				}
			}
		},
		// 验证单个输入框
		validateSingleInput(index){
			var vm = this;
			var filter = vm.nameContainsContentList[index];
			
			if(vm.addRuleForm.enable == '1' && vm.addRuleForm.matching_mode == 'deviceName'){
				// 检查是否为空或格式不正确
				if(!filter.value || filter.value.trim() === ''){
					vm.$set(filter, 'hasError', true);
				} else {
					vm.$set(filter, 'hasError', false);
				}
			} else {
				vm.$set(filter, 'hasError', false);
			}
			
            vm.$refs.addRuleForm.validateField('name_contains');
		},
        //TAC验证
        isValidIntegerTacRange(val, min, max) {
            var vm = this;
            // 将输入字符串按逗号分割成多个部分
            const parts = val.split(',');

            for (let part of parts) {
                // 检查是否有连字符
                if (part.includes('-')) {
                    const range = part.split('-');
                    const start = parseInt(range[0], 10);
                    const end = parseInt(range[1], 10);

                    // 检查范围内的数字是否合法
                    if (isNaN(start) || isNaN(end) || !vm.isNumeric(start) || !vm.isNumeric(end) || start >= end || start < min || end > max) {
                        return false;
                    }
                } else {
                    const num = parseInt(part, 10);

                    // 检查单个数字是否合法
                    if (isNaN(num) || num < min || num > max) {
                        return false;
                    }
                }
            }

            return true;
        },
        // 验证输入的是否是数字
        isNumeric(str) {
            if(str.length==0){
                return false;
            }
            for(var i=0;i<str.length;i++){
                if(str.charAt(i)<"0" || str.charAt(i)>"9"){
                    return false;
                }
            }
            return true;  
        },
	},
	mounted(){
		this.init();
	}
	
})

</script> 
